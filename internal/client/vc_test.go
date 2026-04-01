package client

import (
	"strings"
	"testing"
)

func TestBuildTranscriptQuery(t *testing.T) {
	tests := []struct {
		name      string
		opts      TranscriptOptions
		wantParts []string // 期望 query string 包含的部分
		wantNot   []string // 期望 query string 不包含的部分
	}{
		{
			name: "默认参数_speaker和timestamp都为true",
			opts: TranscriptOptions{NeedSpeaker: true, NeedTimestamp: true},
			wantParts: []string{
				"need_speaker=true",
				"need_timestamp=true",
			},
		},
		{
			name: "不需要speaker",
			opts: TranscriptOptions{NeedSpeaker: false, NeedTimestamp: true},
			wantParts: []string{
				"need_speaker=false",
				"need_timestamp=true",
			},
		},
		{
			name: "srt格式",
			opts: TranscriptOptions{NeedSpeaker: true, NeedTimestamp: true, FileFormat: "srt"},
			wantParts: []string{
				"need_speaker=true",
				"need_timestamp=true",
				"file_format=srt",
			},
		},
		{
			name: "txt格式",
			opts: TranscriptOptions{NeedSpeaker: false, NeedTimestamp: false, FileFormat: "txt"},
			wantParts: []string{
				"need_speaker=false",
				"need_timestamp=false",
				"file_format=txt",
			},
		},
		{
			name:    "空格式不拼接file_format",
			opts:    TranscriptOptions{NeedSpeaker: true, NeedTimestamp: true, FileFormat: ""},
			wantNot: []string{"file_format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTranscriptQuery(tt.opts)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("buildTranscriptQuery() = %q, 缺少 %q", got, part)
				}
			}
			for _, part := range tt.wantNot {
				if strings.Contains(got, part) {
					t.Errorf("buildTranscriptQuery() = %q, 不应包含 %q", got, part)
				}
			}
		})
	}
}

func TestParseTranscriptErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "参数错误",
			body:    []byte(`{"code":2091001,"msg":"param is invalid"}`),
			wantErr: true,
			errMsg:  "2091001",
		},
		{
			name:    "资源不存在",
			body:    []byte(`{"code":2091002,"msg":"resource not found"}`),
			wantErr: true,
			errMsg:  "resource not found",
		},
		{
			name:    "非JSON内容返回nil",
			body:    []byte("这是一段纯文本妙记内容"),
			wantErr: false,
		},
		{
			name:    "code为0的JSON返回nil",
			body:    []byte(`{"code":0,"msg":"success"}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseTranscriptError(tt.body)
			if tt.wantErr {
				if err == nil {
					t.Error("parseTranscriptError() 应返回错误")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseTranscriptError() error = %q, 期望包含 %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("parseTranscriptError() 不应返回错误, got: %v", err)
				}
			}
		})
	}
}

func TestExportMinuteTranscript_NoClient(t *testing.T) {
	// 未配置客户端时应返回错误
	resetClient()
	resetConfig()

	_, err := ExportMinuteTranscript("obcntest", TranscriptOptions{}, "")
	if err == nil {
		t.Error("ExportMinuteTranscript() 未配置时应返回错误")
	}
}
