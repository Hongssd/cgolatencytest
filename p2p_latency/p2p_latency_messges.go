package p2p_latency

type P2PReqType string

const (
	P2PReqTypeLatency   P2PReqType = "latency"    //网络延迟消息
	P2PReqTypeBnLatency P2PReqType = "bn_latency" //币安延迟消息
)

type P2PMessage struct {
	IsReq bool
	Req   P2PReq
	Res   P2PRes
}

// 请求结构
type P2PReq struct {
	ReqId   string
	ReqType P2PReqType
	ReqData string
}

// 响应结构
type P2PRes struct {
	ReqId        string
	ReqType      P2PReqType
	ResData      string
	ErrCode      int
	ErrMsg       string
	InTimestamp  int64
	OutTimestamp int64
}

