package tourismdisputeresolution

import "time"

// ExecutionKind 表示和解结论中需要落地执行的安排种类。
type ExecutionKind string

const (
    ExecRefund     ExecutionKind = "refund"     // 退款
    ExecVoucher    ExecutionKind = "voucher"    // 代金安排
    ExecConclusion ExecutionKind = "conclusion" // 书面结论
)

// ExecutionRecord 记录一项应执行安排及其执行情况。
type ExecutionRecord struct {
    ID        string        `json:"id"`
    Kind      ExecutionKind `json:"kind"`
    Party     string        `json:"party"` // 执行方
    Expected  Money         `json:"expected"`
    Actual    Money         `json:"actual"`
    DueAt     time.Time     `json:"due_at"`
    DoneAt    *time.Time    `json:"done_at,omitempty"`
    Reference string        `json:"reference"` // 支付流水号或结论文号
}

// ExecutionGap 表示核对发现的一项执行落差。
type ExecutionGap struct {
    ExecutionID string `json:"execution_id"`
    Problem     string `json:"problem"` // 未执行、金额不符、币种不符、逾期
}

// VerifyExecution 核对所有退款、代金安排和书面结论是否真正执行到位。
func VerifyExecution(records []ExecutionRecord, now time.Time) []ExecutionGap {
    var gaps []ExecutionGap
    for _, rec := range records {
        if rec.DoneAt == nil {
            if now.After(rec.DueAt) {
                gaps = append(gaps, ExecutionGap{ExecutionID: rec.ID, Problem: "超过期限仍未执行"})
            } else {
                gaps = append(gaps, ExecutionGap{ExecutionID: rec.ID, Problem: "尚未执行"})
            }
            continue
        }
        if rec.Actual.Currency != rec.Expected.Currency {
            gaps = append(gaps, ExecutionGap{ExecutionID: rec.ID, Problem: "执行币种与约定不符"})
            continue
        }
        if rec.Actual.Amount != rec.Expected.Amount {
            gaps = append(gaps, ExecutionGap{ExecutionID: rec.ID, Problem: "执行金额与约定不符"})
            continue
        }
        if rec.DoneAt.After(rec.DueAt) {
            gaps = append(gaps, ExecutionGap{ExecutionID: rec.ID, Problem: "执行逾期"})
        }
    }
    return gaps
}
