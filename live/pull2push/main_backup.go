package main

// Go 1.21+
// HLS Pull -> Republish (multi-stream) in-memory ring buffer
// 功能：
//   - 从上游 HLS（m3u8）持续拉流，抓取新增分片（TS 或 fMP4）
//   - 在本地以新的播放地址 /live/{name}/index.m3u8 重新发布
//   - 支持多路上游（并发抓取），每路维护滑动窗口（可配置分片数量）
//   - 简化实现：不做解码重封装，分片原样转发（最稳定、性能最好）
//   - 适合做“反向代理 / 旁路转推 / 内网汇聚”
//   - 仅演示用途：未实现 AES-128/SAMPLE-AES 解密、LL-HLS、广告标记等高级特性
//
// 运行：
//
//	go mod init hls-relay
//	go get github.com/grafov/m3u8
//	go run . -addr :8080 -buffer 6 \
//	  -src cam=http://example.com/live/cam/playlist.m3u8 \
//	  -src news=https://cdn.example.com/news/stream.m3u8
//
// 播放：
//
//	http://127.0.0.1:8080/live/cam/index.m3u8
//	http://127.0.0.1:8080/live/news/index.m3u8
//
// 生产提示：
//   - 如果需要持久化或大规模并发，建议把分片落地到磁盘并配合 Nginx/静态文件服务
//   - 接入 CDN 时，确保 /index.m3u8 和分片缓存头正确（Cache-Control/ETag 等）
//   - 如需低延迟，缩短上游分片时长并增加轮询频率；若要 LL-HLS，需要 partial segments 支持

/*
使用ffmpeg推流：ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream

拉流
● RTMP (by VLC): rtmp://192.168.203.182/live/livestream
● H5(HTTP-FLV): http://192.168.203.182:8080/live/livestream.flv
● H5(HLS): http://192.168.203.182:8080/live/livestream.m3u8



ffmpeg -re -i demo.flv \
   -c:v libx264 -preset veryfast -tune zerolatency \
   -g 25 -keyint_min 25 \
   -c:a aac -ar 44100 -b:a 128k \
   -f flv rtmp://192.168.203.182/live/livestream

go run . -addr :8080 -buffer 6 -src test=http://192.168.203.182:8080/live/livestream.m3u8

*/

import (
	"bytes"
	"container/ring"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
)

/*
整体功能概述
功能目标：
	从多个上游 HLS 流地址拉流（m3u8 + ts），
	缓存最近若干个分片（ring buffer），
	并通过本地 HTTP 服务将这些分片和对应的 m3u8 播放列表转发出去，
	支持多路同时拉取和复用。

多路复用实现
	关键是用全局 map[string]*StreamState 管理多路拉流状态，路由里根据 URL 路径选择对应流。
	每一路独立拉取并缓存，HTTP 根据 URL 返回对应的 m3u8 和分片。
	这样前端可以访问 /live/cam/index.m3u8、/live/news/index.m3u8，不同流完全隔离。

主要流程：拉流 + 缓存 + HTTP 服务


*/

// Source 每一个直播地址的信息
type Source struct {
	Name    string
	URL     string
	Variant string // 可选：固定选择带宽 id/分辨率（留空自动选最优）
}

// Segment 每一个m3u8数据分片的数据对象，代表 HLS 的一个 TS 或 fMP4 分片。记录了下载地址和本地暴露的名字，数据内容和时长。
type Segment struct {
	Seq       uint64    // 分片序列号（递增）
	URI       string    // 上游绝对地址（下载用）
	LocalName string    // 本地暴露的文件名（如 seq.ts 或 seq.m4s）
	Data      []byte    // 分片字节
	Dur       float64   // 分片时长，秒
	Discont   bool      // 是否断点分片
	AddedAt   time.Time // 拉取时间
}

// StreamState 每一路拉流任务维护一个 StreamState，存放它的分片缓存和元数据。
// 用 ring.Ring 实现固定容量的循环队列，保持缓存窗口。
type StreamState struct {
	Name      string
	Mu        sync.RWMutex
	Segments  *ring.Ring // 环形缓冲，存放最近 N 个分片，元素为 *Segment 或 nil
	Cap       int        // 缓冲分片数
	TargetDur float64    // HLS 目标分片时长
	SeqStart  uint64     // 本地播放列表起始序列号
	LastSeq   uint64     // 最新分片序列号（递增）
	LastMod   time.Time  // 最后更新时间
	Discont   bool       // 是否有断点续播
}

// NewStreamState 创建每一个直播的拉流缓冲区对象
func NewStreamState(name string, cap int) *StreamState {
	return &StreamState{
		Name:      name,
		Segments:  ring.New(cap),
		Cap:       cap,
		TargetDur: 6,
		SeqStart:  0,
		LastSeq:   0,
	}
}

// PushSegment 在环形缓冲里追加一个分片
func (s *StreamState) PushSegment(seg *Segment) {
	/*
		维护固定容量缓存，最新的分片覆盖最旧的。
		更新本地播放列表序列号区间。
		保护并发安全（互斥锁）。
	*/

	s.Mu.Lock()
	defer s.Mu.Unlock()

	// 移动指针到下一格并覆盖
	s.Segments = s.Segments.Next()
	s.Segments.Value = seg

	if s.SeqStart == 0 {
		// 第一次写入
		s.SeqStart = seg.Seq
	}
	// 计算当前窗口的“起始序列号” = 最新序列 - (cap-1)
	if seg.Seq+1 >= uint64(s.Cap) {
		s.SeqStart = seg.Seq + 1 - uint64(s.Cap)
	}
	s.LastSeq = seg.Seq
	s.LastMod = time.Now()
	if seg.Discont {
		s.Discont = true
	}
}

// Snapshot 返回按序的窗口分片拷贝（只读）
func (s *StreamState) Snapshot() (segs []*Segment, seqStart uint64, targetDur float64, discont bool) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	segs = make([]*Segment, 0, s.Cap)
	// 从环形缓冲按时间顺序读出
	tmp := s.Segments
	tmp.Do(func(v any) {
		if v == nil {
			return
		}
		seg := v.(*Segment)
		// 只保留窗口内的、非空数据
		if seg != nil && len(seg.Data) > 0 {
			segs = append(segs, seg)
		}
	})
	return segs, s.SeqStart, s.TargetDur, s.Discont
}

// Global registry of streams
var (
	streamsMu sync.RWMutex
	streams   = map[string]*StreamState{}
)

func getOrCreateStream(name string, cap int) *StreamState {
	streamsMu.Lock()
	defer streamsMu.Unlock()
	if s, ok := streams[name]; ok {
		return s
	}
	s := NewStreamState(name, cap)
	streams[name] = s
	return s
}

// ---------- HLS 拉流逻辑 ----------

// resolveURL 处理相对 URI -> 绝对 URL
func resolveURL(base, ref string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return u.ResolveReference(r).String(), nil
}

// pickVariant 选择主清单里最合适的变体
func pickVariant(master *m3u8.MasterPlaylist, prefer string) (*m3u8.Variant, error) {
	if len(master.Variants) == 0 {
		return nil, errors.New("no variants in master playlist")
	}
	// 优先匹配名称/分辨率/带宽包含 prefer 的条目
	if prefer != "" {
		for _, v := range master.Variants {
			label := []string{v.Name, v.Codecs}
			if v.Resolution != "" {
				label = append(label, v.Resolution)
			}
			label = append(label, strconv.FormatInt(int64(v.Bandwidth), 10))
			joined := strings.ToLower(strings.Join(label, ","))
			if strings.Contains(joined, strings.ToLower(prefer)) {
				return v, nil
			}
		}
	}
	// 否则选带宽最高的
	var best *m3u8.Variant
	for _, v := range master.Variants {
		if best == nil || v.Bandwidth > best.Bandwidth {
			best = v
		}
	}
	return best, nil
}

// fetchOnce 拉取并解析一个 m3u8 文本
func fetchOnce(ctx context.Context, client *http.Client, u string) (m m3u8.Playlist, body []byte, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "hls-relay/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("bad status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	p, listType, err := m3u8.Decode(*bytes.NewBuffer(b), true)
	if err != nil {
		return nil, nil, err
	}
	_ = listType
	return p, b, nil
}

// pullWorker 持续从上游拉取分片并写入 stream state
func pullWorker(ctx context.Context, src Source, bufCount int) {
	/*
		这是一个后台 goroutine，用来持续从某个 HLS 上游地址拉取数据。
		它先请求 Master Playlist，如果是多码率流，选择合适变体变成 Media Playlist。
		定时轮询 Media Playlist（默认 800ms），发现新分片后下载。
		下载到分片后，调用 stream.PushSegment() 把它放入对应的 StreamState 环形缓存。

		核心点：
			通过 seen 维护已下载分片，避免重复下载。
			计算本地序列号 Seq，保证分片顺序。
			下载的分片保持原样字节，不做解码重封装，性能好且稳定。
	*/

	log.Printf("[pull:%s] start from %s", src.Name, src.URL)
	client := &http.Client{Timeout: 10 * time.Second}

	stream := getOrCreateStream(src.Name, bufCount)
	seen := map[string]bool{}
	var mediaURL string

	// 初次处理 master/ media
	p, _, err := fetchOnce(ctx, client, src.URL)
	if err != nil {
		log.Printf("[pull:%s] fetch master/media failed: %v", src.Name, err)
		return
	}
	if mp, ok := p.(*m3u8.MasterPlaylist); ok {
		v, err := pickVariant(mp, src.Variant)
		if err != nil {
			log.Printf("[pull:%s] no variant: %v", src.Name, err)
			return
		}
		mediaURL, err = resolveURL(src.URL, v.URI)
		if err != nil {
			log.Printf("[pull:%s] resolve media url: %v", src.Name, err)
			return
		}
		log.Printf("[pull:%s] choose variant bw=%d res=%s uri=%s", src.Name, v.Bandwidth, v.Resolution, mediaURL)
	} else if _, ok := p.(*m3u8.MediaPlaylist); ok {
		mediaURL = src.URL
	} else {
		log.Printf("[pull:%s] unknown playlist type", src.Name)
		return
	}

	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	var lastSeq uint64
	for {
		select {
		case <-ctx.Done():
			log.Printf("[pull:%s] stop", src.Name)
			return
		case <-ticker.C:
			p, _, err := fetchOnce(ctx, client, mediaURL)
			if err != nil {
				log.Printf("[pull:%s] fetch media: %v", src.Name, err)
				continue
			}
			mp, ok := p.(*m3u8.MediaPlaylist)
			if !ok {
				log.Printf("[pull:%s] not media playlist", src.Name)
				continue
			}

			// 更新 target duration
			if mp.TargetDuration > 0 {
				stream.Mu.Lock()
				stream.TargetDur = float64(mp.TargetDuration)
				stream.Mu.Unlock()
			}

			// 遍历新片段
			for _, seg := range mp.Segments {
				if seg == nil {
					continue
				}
				absURI, err := resolveURL(mediaURL, seg.URI)
				if err != nil {
					continue
				}
				if seen[absURI] {
					continue
				}

				// 估算 seq：用节目序列号 + 相对偏移（若提供）
				var seq uint64
				if mp.SeqNo != 0 {
					seq = uint64(mp.SeqNo) + uint64(seg.SeqId)
				} else {
					// 回退：自增
					lastSeq++
					seq = lastSeq
				}

				data, err := download(ctx, client, absURI)
				if err != nil {
					log.Printf("[pull:%s] seg dl: %v", src.Name, err)
					continue
				}

				localName := localSegName(absURI, seq)
				stream.PushSegment(&Segment{
					Seq:       seq,
					URI:       absURI,
					LocalName: localName,
					Data:      data,
					Dur:       seg.Duration,
					Discont:   seg.Discontinuity,
					AddedAt:   time.Now(),
				})

				seen[absURI] = true
				lastSeq = seq
			}
		}
	}
}

func download(ctx context.Context, client *http.Client, u string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "hls-relay/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status %d for %s", resp.StatusCode, u)
	}
	return io.ReadAll(resp.Body)
}

func localSegName(absURI string, seq uint64) string {
	// 统一以 seq+原始后缀 命名，便于本地播放器顺序请求
	u, err := url.Parse(absURI)
	if err != nil {
		return fmt.Sprintf("%d.bin", seq)
	}
	ext := path.Ext(u.Path)
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%d%s", seq, ext)
}

// ---------- HTTP 服务 ----------

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// /live/{name}/index.m3u8
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "live" || parts[2] != "index.m3u8" {
		http.NotFound(w, r)
		return
	}
	name := parts[1]
	streamsMu.RLock()
	s, ok := streams[name]
	streamsMu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	segs, seqStart, targetDur, discont := s.Snapshot()
	pl, err := buildMediaPlaylist(segs, seqStart, targetDur, discont, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(pl))
}

func handleSegment(w http.ResponseWriter, r *http.Request) {
	// /live/{name}/{seg.ts|m4s}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "live" {
		http.NotFound(w, r)
		return
	}
	name := parts[1]
	filename := parts[2]

	streamsMu.RLock()
	s, ok := streams[name]
	streamsMu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	s.Mu.RLock()
	defer s.Mu.RUnlock()

	var seg *Segment
	s.Segments.Do(func(v any) {
		if v == nil {
			return
		}
		ss := v.(*Segment)
		if ss != nil && ss.LocalName == filename {
			seg = ss
		}
	})
	if seg == nil {
		http.NotFound(w, r)
		return
	}

	// 内容类型根据后缀猜测
	if strings.HasSuffix(filename, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	} else if strings.HasSuffix(filename, ".m4s") || strings.HasSuffix(filename, ".mp4") {
		w.Header().Set("Content-Type", "video/mp4")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	_, _ = w.Write(seg.Data)
}

// buildMediaPlaylist HTTP 播放列表生成与分片访问
// 生成 HLS 标准的分片信息，顺序排列，标明时长和断点。
// 返回给播放器标准 HLS 播放列表。
func buildMediaPlaylist(segs []*Segment, seqStart uint64, targetDur float64, discont bool, r *http.Request) (string, error) {

	// handleIndex 负责根据当前 StreamState 缓存的分片快照，生成标准 HLS 播放列表文本。
	// 播放列表里指向本地缓存的分片文件名（seq.ts 或 seq.m4s）。
	// handleSegment 负责根据请求的分片名返回对应的分片字节流。

	if len(segs) == 0 {
		// 空列表也要有基本头信息，避免播放器报错
		return "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:0\n", nil
	}
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	b.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(targetDur+0.5)))
	b.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", seqStart))
	// 可选：I-Frame only、MAP 等根据上游情况补充
	if discont {
		b.WriteString("#EXT-X-DISCONTINUITY-SEQUENCE:1\n")
	}

	base := fmt.Sprintf("/live/%s/", strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[1])
	for _, s := range segs {
		if s == nil {
			continue
		}
		if s.Discont {
			b.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		b.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", s.Dur))
		b.WriteString(base + s.LocalName + "\n")
	}
	return b.String(), nil
}

// ---------- main ----------

func main() {
	//buffer := flag.Int("buffer", 6, "per-stream segment buffer size (playlist window)")
	buffer := 6
	var srcs multiSourceFlag
	flag.Var(&srcs, "src", "add a source: name=url or name=url,variant=720")
	flag.Parse()

	if len(srcs) == 0 {
		log.Println("用法示例：\n  go run . -addr :8080 -buffer 6 -src cam=http://example/live/cam.m3u8 -src news=https://cdn/news.m3u8")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, s := range srcs {
		go pullWorker(ctx, s, buffer)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(gin.Recovery())

	// 添加 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "android-device-id", "Content-Type", "Accept", "Authorization", "X-Token", "Device-Id", "request-time", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// health
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/live/*filepath", func(c *gin.Context) {
		path1 := c.Param("filepath") // 例如 "/test/index.m3u8"
		if strings.HasSuffix(path1, "/index.m3u8") {
			handleIndex(c.Writer, c.Request)
			return
		}
		handleSegment(c.Writer, c.Request)
	})

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// 解析 -src 参数
// 形式： name=url 或 name=url,variant=720p

type multiSourceFlag []Source

func (m *multiSourceFlag) String() string {
	var parts []string
	for _, s := range *m {
		parts = append(parts, fmt.Sprintf("%s=%s", s.Name, s.URL))
	}
	return strings.Join(parts, ",")
}

func (m *multiSourceFlag) Set(v string) error {
	// 拆 name=rest
	kv := strings.SplitN(v, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("invalid src: %s", v)
	}
	name := kv[0]
	rest := kv[1]

	urlPart := rest
	variant := ""
	// 支持逗号后追加键值
	if strings.Contains(rest, ",") {
		pieces := strings.Split(rest, ",")
		urlPart = pieces[0]
		for _, p := range pieces[1:] {
			if strings.HasPrefix(p, "variant=") {
				variant = strings.TrimPrefix(p, "variant=")
			}
		}
	}
	if !strings.HasPrefix(urlPart, "http") {
		return fmt.Errorf("invalid url in src: %s", urlPart)
	}
	*m = append(*m, Source{Name: name, URL: urlPart, Variant: variant})
	return nil
}
