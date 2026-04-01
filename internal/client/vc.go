package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const vcBase = "/open-apis/vc/v1"

// ==================== 会议列表 ====================

// SearchMeetings 搜索会议列表
// 使用 GET /open-apis/vc/v1/meeting_list 获取会议列表
func SearchMeetings(startTime, endTime int64, meetingStatus int, meetingNo string, pageSize int, pageToken string, userAccessToken string) (json.RawMessage, string, bool, error) {
	client, err := GetClient()
	if err != nil {
		return nil, "", false, err
	}

	apiPath := fmt.Sprintf("%s/meeting_list?start_time=%s&end_time=%s",
		vcBase,
		strconv.FormatInt(startTime, 10),
		strconv.FormatInt(endTime, 10),
	)

	if meetingStatus > 0 {
		apiPath += fmt.Sprintf("&meeting_status=%d", meetingStatus)
	}
	if meetingNo != "" {
		apiPath += "&meeting_no=" + meetingNo
	}
	if pageSize > 0 {
		apiPath += fmt.Sprintf("&page_size=%d", pageSize)
	}
	if pageToken != "" {
		apiPath += "&page_token=" + pageToken
	}

	tokenType, opts := resolveTokenOpts(userAccessToken)

	resp, err := client.Get(Context(), apiPath, nil, tokenType, opts...)
	if err != nil {
		return nil, "", false, fmt.Errorf("搜索会议列表失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", false, fmt.Errorf("搜索会议列表失败: HTTP %d, body: %s", resp.StatusCode, string(resp.RawBody))
	}

	var apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MeetingList json.RawMessage `json:"meeting_list"`
			PageToken   string          `json:"page_token"`
			HasMore     bool            `json:"has_more"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.RawBody, &apiResp); err != nil {
		return nil, "", false, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, "", false, fmt.Errorf("搜索会议列表失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	return apiResp.Data.MeetingList, apiResp.Data.PageToken, apiResp.Data.HasMore, nil
}

// ==================== 会议详情 ====================

// GetMeeting 获取会议详情
// 使用 GET /open-apis/vc/v1/meetings/:meeting_id
func GetMeeting(meetingID string, userAccessToken string) (json.RawMessage, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	tokenType, opts := resolveTokenOpts(userAccessToken)

	apiPath := fmt.Sprintf("%s/meetings/%s", vcBase, meetingID)
	resp, err := client.Get(Context(), apiPath, nil, tokenType, opts...)
	if err != nil {
		return nil, fmt.Errorf("获取会议详情失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取会议详情失败: HTTP %d, body: %s", resp.StatusCode, string(resp.RawBody))
	}

	var apiResp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(resp.RawBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("获取会议详情失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	return apiResp.Data, nil
}

// ==================== 妙记 ====================

// GetMinute 获取妙记信息
// 使用 GET /open-apis/minutes/v1/minutes/:minute_token
func GetMinute(minuteToken string, userAccessToken string) (json.RawMessage, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	tokenType, opts := resolveTokenOpts(userAccessToken)

	apiPath := fmt.Sprintf("/open-apis/minutes/v1/minutes/%s", minuteToken)
	resp, err := client.Get(Context(), apiPath, nil, tokenType, opts...)
	if err != nil {
		return nil, fmt.Errorf("获取妙记信息失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取妙记信息失败: HTTP %d, body: %s", resp.StatusCode, string(resp.RawBody))
	}

	var apiResp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(resp.RawBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("获取妙记信息失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	return apiResp.Data, nil
}

// ==================== 妙记文字记录 ====================

// TranscriptOptions 导出妙记文字记录的选项
type TranscriptOptions struct {
	NeedSpeaker   bool   // 是否包含说话人
	NeedTimestamp bool   // 是否包含时间戳
	FileFormat    string // 导出格式：txt 或 srt，空则不传
}

// buildTranscriptQuery 构建 transcript 接口的查询参数
func buildTranscriptQuery(opts TranscriptOptions) string {
	q := fmt.Sprintf("need_speaker=%t&need_timestamp=%t", opts.NeedSpeaker, opts.NeedTimestamp)
	if opts.FileFormat != "" {
		q += "&file_format=" + opts.FileFormat
	}
	return q
}

// parseTranscriptError 尝试将响应体解析为 JSON 错误，非 JSON 内容返回 nil。
// API 成功时返回非 JSON 的文本流，code==0 的 JSON 理论上不会出现，此处兜底返回 nil。
func parseTranscriptError(body []byte) error {
	var apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// 非 JSON 内容，说明是正常的文本流
		return nil
	}
	if apiResp.Code != 0 {
		return fmt.Errorf("导出妙记文字记录失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}
	return nil
}

// ExportMinuteTranscript 导出妙记文字记录
// GET /open-apis/minutes/v1/minutes/:minute_token/transcript
// 成功时返回文件二进制流（txt/srt），失败时返回 JSON 错误
func ExportMinuteTranscript(minuteToken string, opts TranscriptOptions, userAccessToken string) ([]byte, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	tokenType, reqOpts := resolveTokenOpts(userAccessToken)

	apiPath := fmt.Sprintf("/open-apis/minutes/v1/minutes/%s/transcript?%s", minuteToken, buildTranscriptQuery(opts))
	resp, err := client.Get(Context(), apiPath, nil, tokenType, reqOpts...)
	if err != nil {
		return nil, fmt.Errorf("导出妙记文字记录失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("导出妙记文字记录失败: HTTP %d, body: %s", resp.StatusCode, string(resp.RawBody))
	}

	// 检查响应体是否为 JSON 错误
	if err := parseTranscriptError(resp.RawBody); err != nil {
		return nil, err
	}

	return resp.RawBody, nil
}
