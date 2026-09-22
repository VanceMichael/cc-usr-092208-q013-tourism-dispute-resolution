package tourismdisputeresolution

import (
    "fmt"
    "time"
)

// EventKind 表示协商过程中需要保持清晰顺序的事件种类。
type EventKind string

const (
    EventCaseOpened         EventKind = "case_opened"         // 立案
    EventCompanionSplit     EventKind = "companion_split"     // 同行人拆分
    EventPartialPerformance EventKind = "partial_performance" // 部分履约确认
    EventEvidenceSupplement EventKind = "evidence_supplement" // 证据补交
    EventDeadlineSuspended  EventKind = "deadline_suspended"  // 时限暂停
    EventDeadlineResumed    EventKind = "deadline_resumed"    // 时限恢复
    EventMediationFailed    EventKind = "mediation_failed"    // 调解失败
    EventPaymentReversed    EventKind = "payment_reversed"    // 支付退回
    EventRefundExecuted     EventKind = "refund_executed"     // 退款执行
    EventVoucherIssued      EventKind = "voucher_issued"      // 代金安排
    EventConclusionIssued   EventKind = "conclusion_issued"   // 书面结论
)

// CaseEvent 表示协商时间线中的一个有序事件。
type CaseEvent struct {
    Seq        int       `json:"seq"` // 从 1 开始连续编号
    Kind       EventKind `json:"kind"`
    ItemID     string    `json:"item_id,omitempty"`
    Actor      string    `json:"actor"`
    OccurredAt time.Time `json:"occurred_at"`
    Detail     string    `json:"detail"`
}

// ValidateEventOrder 确认事件编号从 1 开始连续递增，且发生时间不倒退，
// 使同行人拆分、证据补交、时限暂停、调解失败、支付退回等步骤保持清晰顺序。
func ValidateEventOrder(events []CaseEvent) error {
    for i, ev := range events {
        if ev.Seq != i+1 {
            return fmt.Errorf("事件顺序不连续: 第 %d 位事件编号为 %d", i+1, ev.Seq)
        }
        if ev.Kind == "" || ev.Actor == "" {
            return fmt.Errorf("事件 %d 缺少种类或行为方", ev.Seq)
        }
        if i > 0 && ev.OccurredAt.Before(events[i-1].OccurredAt) {
            return fmt.Errorf("事件 %d 的发生时间早于前一事件", ev.Seq)
        }
    }
    return nil
}
