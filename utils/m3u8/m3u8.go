package m3u8

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

// PlaylistType 定义播放列表类型
type PlaylistType string

const (
	MASTER  PlaylistType = "MASTER"
	VOD     PlaylistType = "VOD"
	LIVE    PlaylistType = "LIVE"
	EVENT   PlaylistType = "EVENT"
	UNKNOWN PlaylistType = "UNKNOWN"
)

// StreamInfo 存储流信息
type StreamInfo struct {
	Name       string
	Bandwidth  int
	Resolution string
	Codecs     string
	URL        string
}

// Info 存储M3U8解析信息
type Info struct {
	Type             PlaylistType
	Version          int
	StreamCount      int
	Streams          []StreamInfo
	SegmentCount     int
	TotalDuration    float64
	TargetDuration   float64
	MediaSequence    int
	HasEndList       bool
	HasDiscontinuity bool
	HasUnknownTags   bool
	Segments         []string // TS片段列表
}

// ParseM3U8 解析M3U8内容
func ParseM3U8(client *http.Client, Url, UserAgent, Cookie, Origin, Referer string) (*Info, error) {
	Url = strings.Replace(Url, "\r\n", "", -1)
	Url = strings.Replace(Url, "\r", "", -1)
	Url = strings.Replace(Url, "\n", "", -1)
	var err error
	var req *http.Request
	req, err = http.NewRequest("GET", Url, nil)
	if err != nil {
		return nil, err
	}
	if len(UserAgent) > 0 {
		req.Header.Set("User-Agent", UserAgent)
	}
	if len(Cookie) > 0 {
		req.Header.Set("Cookie", Cookie)
	}
	if len(Origin) > 0 {
		req.Header.Set("Origin", Origin)
	}
	if len(Referer) > 0 {
		req.Header.Set("Referer", Referer)
	}
	var body []byte
	var response *http.Response
	response, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	body, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	content := string(body)
	u, _ := url.Parse(Url)
	baseUrl := u.Scheme + "://" + u.Host + strings.Replace(filepath.Dir(u.EscapedPath()), "\\", "/", -1) + "/"
	info := &Info{
		Type:     UNKNOWN,
		Streams:  make([]StreamInfo, 0),
		Segments: make([]string, 0),
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	// 检查是否以#EXTM3U开头
	if !scanner.Scan() || scanner.Text() != "#EXTM3U" {
		return info, nil
	}
	var isMaster, isVOD, isEvent bool
	var currentStream *StreamInfo
	var durations []float64
	var validTagCount int // 统计有效标签数量
	for scanner.Scan() {
		line := scanner.Text()
		// 处理不同的标签
		switch {
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:"):
			isMaster = true
			validTagCount++
			currentStream = &StreamInfo{}
			// 解析流参数
			params := strings.Split(line[18:], ",")
			for _, param := range params {
				if strings.HasPrefix(param, "BANDWIDTH=") {
					bw, _ := strconv.Atoi(param[10:])
					currentStream.Bandwidth = bw
				} else if strings.HasPrefix(param, "RESOLUTION=") {
					currentStream.Resolution = param[11:]
				} else if strings.HasPrefix(param, "CODECS=") {
					currentStream.Codecs = param[7:]
				} else if strings.HasPrefix(param, "NAME=") {
					currentStream.Name = param[5:]
				}
			}

		case strings.HasPrefix(line, "#EXT-X-VERSION:"):
			validTagCount++
			version, _ := strconv.Atoi(line[16:])
			info.Version = version

		case line == "#EXT-X-PLAYLIST-TYPE:VOD":
			validTagCount++
			isVOD = true
			break
		case strings.HasPrefix(line, "#EXT-X-TARGETDURATION:"):
			validTagCount++
			duration, _ := strconv.ParseFloat(line[23:], 64)
			info.TargetDuration = duration
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			validTagCount++
			seq, _ := strconv.Atoi(line[23:])
			info.MediaSequence = seq
		case strings.HasPrefix(line, "#EXTINF:"):
			validTagCount++
			if !isMaster {
				durationStr := strings.TrimSuffix(line[8:], ",")
				duration, _ := strconv.ParseFloat(durationStr, 64)
				durations = append(durations, duration)
			}
		case line == "#EXT-X-ENDLIST":
			validTagCount++
			info.HasEndList = true
		case line == "#EXT-X-DISCONTINUITY":
			validTagCount++
			isEvent = true
			info.HasDiscontinuity = true
		case strings.HasPrefix(line, "#EXT-X-"):
			// 只统计标准的M3U8标签
			if !strings.HasPrefix(line, "#EXT-X-STREAM-INF:") &&
				!strings.HasPrefix(line, "#EXT-X-VERSION:") &&
				!strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:") &&
				!strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") &&
				!strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") &&
				!strings.HasPrefix(line, "#EXTINF:") {
				info.HasUnknownTags = true
				validTagCount++ // 未知标签也算作有效标签
			}

		default:
			// 处理URI行
			if currentStream != nil && !strings.HasPrefix(line, "#") {
				validTagCount++
				urls := line
				if !strings.HasPrefix(urls, "http") {
					urls = fmt.Sprintf("%s/%s", baseUrl, urls)
					u, _ = url.Parse(urls)
					u.Path = strings.ReplaceAll(u.Path, "//", "/")
					urls = u.String()
				}
				currentStream.URL = urls
				info.Streams = append(info.Streams, *currentStream)
				currentStream = nil
			} else if !strings.HasPrefix(line, "#") && line != "" {
				// 收集TS片段，但只在有相关标签的情况下才算有效内容
				if validTagCount > 0 { // 只有前面已经有有效标签时，片段才算有效
					urls := line
					if !strings.HasPrefix(urls, "http") {
						urls = fmt.Sprintf("%s/%s", baseUrl, urls)
						u, _ = url.Parse(urls)
						u.Path = strings.ReplaceAll(u.Path, "//", "/")
						urls = u.String()
					}
					info.Segments = append(info.Segments, urls)
				}
			}
		}
	}
	// 只有当发现足够多的有效标签时才设置播放列表类型
	if validTagCount > 0 {
		// 设置播放列表类型
		if isMaster {
			info.Type = MASTER
			info.StreamCount = len(info.Streams)
		} else if isVOD && info.HasEndList {
			info.Type = VOD
			info.SegmentCount = len(durations)
			for _, d := range durations {
				info.TotalDuration += d
			}
		} else if isEvent {
			info.Type = EVENT
			info.SegmentCount = len(durations)
			for _, d := range durations {
				info.TotalDuration += d
			}
		} else if !info.HasEndList && validTagCount > 1 { // 至少需要两个标签才能判定为LIVE
			info.Type = LIVE
			info.SegmentCount = len(durations)
			for _, d := range durations {
				info.TotalDuration += d
			}
		} else if validTagCount > 1 { // 其他情况至少需要两个标签
			// 如果只有#EXTM3U和其他少量内容，则仍视为UNKNOWN
			info.Type = UNKNOWN
		}
	}
	return info, nil
}
