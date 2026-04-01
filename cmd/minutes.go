package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/riba2534/feishu-cli/internal/client"
	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/cobra"
)

// minutesCmd 妙记父命令
var minutesCmd = &cobra.Command{
	Use:   "minutes",
	Short: "妙记操作命令",
	Long: `妙记相关操作，包括获取妙记信息、导出文字记录等。

子命令:
  get         获取妙记信息
  transcript  导出妙记文字记录

示例:
  feishu-cli minutes get obcnxxxx
  feishu-cli minutes transcript obcnxxxx -o output.txt`,
}

var minutesGetCmd = &cobra.Command{
	Use:   "get <minute_token>",
	Short: "获取妙记信息",
	Long: `通过妙记 Token 获取妙记基础信息，包括标题、链接、创建时间、时长等。

参数:
  minute_token  妙记 Token

示例:
  # 获取妙记信息
  feishu-cli minutes get obcnxxxx

  # JSON 格式输出
  feishu-cli minutes get obcnxxxx -o json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return err
		}

		token := resolveOptionalUserToken(cmd)
		minuteToken := args[0]
		output, _ := cmd.Flags().GetString("output")

		data, err := client.GetMinute(minuteToken, token)
		if err != nil {
			return err
		}

		if output == "json" {
			return printJSON(json.RawMessage(data))
		}

		return printMinuteInfo(data)
	},
}

// minutesTranscriptCmd 导出妙记文字记录
var minutesTranscriptCmd = &cobra.Command{
	Use:   "transcript <minute_token>",
	Short: "导出妙记文字记录",
	Long: `导出妙记的对话文本，支持 txt 和 srt 格式。

参数:
  minute_token  妙记 Token

示例:
  # 导出到标准输出
  feishu-cli minutes transcript obcnxxxx

  # 导出为 txt 文件
  feishu-cli minutes transcript obcnxxxx -o output.txt

  # 导出为 srt 字幕文件（含说话人和时间戳）
  feishu-cli minutes transcript obcnxxxx -o output.srt --format srt

  # 不包含说话人
  feishu-cli minutes transcript obcnxxxx --speaker=false`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return err
		}

		token := resolveOptionalUserToken(cmd)
		minuteToken := args[0]
		output, _ := cmd.Flags().GetString("output")
		speaker, _ := cmd.Flags().GetBool("speaker")
		timestamp, _ := cmd.Flags().GetBool("timestamp")
		fileFormat, _ := cmd.Flags().GetString("format")

		// 校验导出格式
		if fileFormat != "" && fileFormat != "txt" && fileFormat != "srt" {
			return fmt.Errorf("不支持的格式 %q，仅支持 txt 或 srt", fileFormat)
		}

		opts := client.TranscriptOptions{
			NeedSpeaker:   speaker,
			NeedTimestamp: timestamp,
			FileFormat:    fileFormat,
		}

		data, err := client.ExportMinuteTranscript(minuteToken, opts, token)
		if err != nil {
			return err
		}

		// 输出到文件或 stdout
		if output != "" {
			if err := os.WriteFile(output, data, 0644); err != nil {
				return fmt.Errorf("写入文件失败: %w", err)
			}
			fmt.Printf("已导出到 %s（%d 字节）\n", output, len(data))
			return nil
		}

		// 输出到 stdout
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(minutesCmd)
	minutesCmd.AddCommand(minutesGetCmd)
	minutesCmd.AddCommand(minutesTranscriptCmd)
	minutesGetCmd.Flags().StringP("output", "o", "", "输出格式（json）")
	minutesGetCmd.Flags().String("user-access-token", "", "User Access Token（用户授权令牌）")

	minutesTranscriptCmd.Flags().StringP("output", "o", "", "输出文件路径")
	minutesTranscriptCmd.Flags().Bool("speaker", true, "包含说话人（默认 true）")
	minutesTranscriptCmd.Flags().Bool("timestamp", true, "包含时间戳（默认 true）")
	minutesTranscriptCmd.Flags().String("format", "", "导出格式：txt 或 srt（不指定则由 API 决定）")
	minutesTranscriptCmd.Flags().String("user-access-token", "", "User Access Token（用户授权令牌）")
}
