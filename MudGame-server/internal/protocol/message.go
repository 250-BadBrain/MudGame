package protocol

import "time"

type CommandMessage struct {
	Type     string                 `json:"type"`
	Subtype  string                 `json:"subtype"`
	PlayerID string                 `json:"playerId,omitempty"`
	Ts       int64                  `json:"ts,omitempty"`
	Args     map[string]interface{} `json:"args,omitempty"`
}

type GameResult struct {
	Type    string                 `json:"type"`
	Subtype string                 `json:"subtype"`
	Flag    bool                   `json:"flag"`
	Ts      int64                  `json:"ts"`
	Results map[string]interface{} `json:"results,omitempty"`
	Logs    []string               `json:"logs,omitempty"`
}

const (
	TypeSystem  = "system"
	TypeCommand = "command"
	TypeChat    = "chat"
	TypeEvent   = "event"
)

func NewResult(msgType, subtype string, ok bool, results map[string]interface{}, logs []string) GameResult {
	return GameResult{
		Type:    msgType,
		Subtype: subtype,
		Flag:    ok,
		Ts:      time.Now().UnixMilli(),
		Results: results,
		Logs:    logs,
	}
}
