package tourismdisputeresolution

import (
    "encoding/json"
    "errors"
    "fmt"
    "time"
)

// Role 表示争议协商中的参与方角色。
type Role string

const (
    RoleTourist  Role = "tourist"  // 入境游客（含同行人）
    RoleOperator Role = "operator" // 境外组团社
    RoleSupplier Role = "supplier" // 国内文旅服务商（地接、景区、承运人）
    RolePayment  Role = "payment"  // 支付机构
    RoleMediator Role = "mediator" // 调解人员
)

// Money 表示一笔带币种的金额。
type Money struct {
    Amount   float64 `json:"amount"`
    Currency string  `json:"currency"`
}

// ServiceItem 表示原合同中的一个服务项。
type ServiceItem struct {
    ID       string `json:"id"`
    Supplier string `json:"supplier"` // 履约企业标识
    Title    string `json:"title"`
    Price    Money  `json:"price"`
}

// Contract 表示争议所依据的原合同。
type Contract struct {
    ID       string        `json:"id"`
    Tourist  string        `json:"tourist"`  // 主游客标识（示例使用化名）
    Operator string        `json:"operator"` // 境外组团社标识
    Items    []ServiceItem `json:"items"`
}

// PerformanceStatus 表示服务项的实际履约状态。
type PerformanceStatus string

const (
    PerformanceFulfilled PerformanceStatus = "fulfilled" // 已履约
    PerformancePartial   PerformanceStatus = "partial"   // 部分履约
    PerformanceCancelled PerformanceStatus = "cancelled" // 因航班变化等原因取消
)

// PerformanceEvent 表示一次实际履约事件。
// Shortfall 为未履约比例（0 到 1）：取消为 1，部分履约按实际缺口填写。
type PerformanceEvent struct {
    ItemID     string            `json:"item_id"`
    Status     PerformanceStatus `json:"status"`
    Shortfall  float64           `json:"shortfall"`
    OccurredAt time.Time         `json:"occurred_at"`
    Note       string            `json:"note"`
}

// ClaimStatus 表示索赔项的协商状态。
type ClaimStatus string

const (
    ClaimOpen        ClaimStatus = "open"        // 待协商
    ClaimNegotiating ClaimStatus = "negotiating" // 协商中
    ClaimSettled     ClaimStatus = "settled"     // 已和解
    ClaimEscalated   ClaimStatus = "escalated"   // 调解失败，转其他途径
    ClaimWithdrawn   ClaimStatus = "withdrawn"   // 已撤回
)

// Claim 表示一名游客（或拆分后的同行人）提出的一笔索赔。
type Claim struct {
    ID       string      `json:"id"`
    Claimant string      `json:"claimant"` // 索赔人标识（同行人各自标识）
    ItemIDs  []string    `json:"item_ids"` // 涉及的合同服务项
    Amount   Money       `json:"amount"`   // 索赔请求金额
    Grounds  string      `json:"grounds"`  // 索赔事由
    Status   ClaimStatus `json:"status"`
}

// Statement 表示一方提交的多语种陈述。
type Statement struct {
    Author   string `json:"author"`
    Language string `json:"language"` // 语种代码，如 zh、ko、en
    Text     string `json:"text"`
}

// EvidenceKind 表示证据材料的种类。
type EvidenceKind string

const (
    EvidenceReceipt      EvidenceKind = "receipt"      // 票据
    EvidenceConfirmation EvidenceKind = "confirmation" // 服务确认
    EvidenceDocument     EvidenceKind = "document"     // 其他书面材料
)

// Evidence 表示一份证据材料。ItemID 为空时不属于任何企业的服务项。
type Evidence struct {
    ID          string       `json:"id"`
    Kind        EvidenceKind `json:"kind"`
    ItemID      string       `json:"item_id,omitempty"`
    SubmittedBy string       `json:"submitted_by"`
    SubmittedAt time.Time    `json:"submitted_at"`
    Supplement  bool         `json:"supplement"` // 是否为补交材料
}

// FXPoint 表示一个汇率时点，用于把退款折算为游客收单币种。
type FXPoint struct {
    Base  string    `json:"base"`  // 原币种
    Quote string    `json:"quote"` // 目标币种
    Rate  float64   `json:"rate"`
    AsOf  time.Time `json:"as_of"`
}

// LiabilityOpinion 表示一方对某服务项发表的责任意见。
type LiabilityOpinion struct {
    Party   string `json:"party"`
    ItemID  string `json:"item_id"`
    Accepts bool   `json:"accepts"` // 是否自认责任
    Note    string `json:"note"`
}

// Case 表示一件旅程争议协商案件的全部归并资料。
type Case struct {
    ID          string             `json:"case_id"`
    Contract    Contract           `json:"contract"`
    Performance []PerformanceEvent `json:"performance"`
    Claims      []Claim            `json:"claims"`
    Statements  []Statement        `json:"statements"`
    Evidence    []Evidence         `json:"evidence"`
    FX          []FXPoint          `json:"fx"`
    Opinions    []LiabilityOpinion `json:"opinions"`
    Events      []CaseEvent        `json:"events"`
    Executions  []ExecutionRecord  `json:"executions"`
    // NextParty 记录每个未决服务项的下一责任方，游客始终可见。
    NextParty map[string]string `json:"next_party"`
}

// ParseCase 读取并检查一件争议案件的归并资料。
func ParseCase(raw []byte) (Case, error) {
    var c Case
    if err := json.Unmarshal(raw, &c); err != nil {
        return Case{}, err
    }
    if err := c.Validate(); err != nil {
        return Case{}, err
    }
    return c, nil
}

// Validate 检查案件资料的结构完整性与引用一致性。
func (c Case) Validate() error {
    if c.ID == "" || c.Contract.ID == "" || c.Contract.Tourist == "" || c.Contract.Operator == "" {
        return errors.New("案件缺少必要标识")
    }
    if len(c.Contract.Items) == 0 {
        return errors.New("原合同缺少服务项")
    }
    items := make(map[string]ServiceItem, len(c.Contract.Items))
    for _, item := range c.Contract.Items {
        if item.ID == "" || item.Supplier == "" || item.Price.Currency == "" {
            return fmt.Errorf("服务项缺少必要字段: %q", item.ID)
        }
        if _, dup := items[item.ID]; dup {
            return fmt.Errorf("服务项标识重复: %s", item.ID)
        }
        items[item.ID] = item
    }
    for _, perf := range c.Performance {
        if _, ok := items[perf.ItemID]; !ok {
            return fmt.Errorf("履约事件引用了未知服务项: %s", perf.ItemID)
        }
        if perf.Shortfall < 0 || perf.Shortfall > 1 {
            return fmt.Errorf("履约缺口比例超出范围: %s", perf.ItemID)
        }
    }
    claims := make(map[string]bool, len(c.Claims))
    for _, claim := range c.Claims {
        if claim.ID == "" || claim.Claimant == "" || len(claim.ItemIDs) == 0 {
            return fmt.Errorf("索赔缺少必要字段: %q", claim.ID)
        }
        if claims[claim.ID] {
            return fmt.Errorf("索赔标识重复: %s", claim.ID)
        }
        claims[claim.ID] = true
        for _, itemID := range claim.ItemIDs {
            if _, ok := items[itemID]; !ok {
                return fmt.Errorf("索赔 %s 引用了未知服务项: %s", claim.ID, itemID)
            }
        }
    }
    for _, ev := range c.Evidence {
        if ev.ID == "" || ev.SubmittedBy == "" {
            return errors.New("证据缺少必要字段")
        }
        if ev.ItemID != "" {
            if _, ok := items[ev.ItemID]; !ok {
                return fmt.Errorf("证据 %s 引用了未知服务项: %s", ev.ID, ev.ItemID)
            }
        }
    }
    for _, fx := range c.FX {
        if fx.Base == "" || fx.Quote == "" || fx.Rate <= 0 {
            return errors.New("汇率时点缺少必要字段")
        }
    }
    for _, op := range c.Opinions {
        if _, ok := items[op.ItemID]; !ok {
            return fmt.Errorf("责任意见引用了未知服务项: %s", op.ItemID)
        }
    }
    if err := ValidateEventOrder(c.Events); err != nil {
        return err
    }
    return nil
}

// Item 按标识查找合同服务项。
func (c Case) Item(itemID string) (ServiceItem, bool) {
    for _, item := range c.Contract.Items {
        if item.ID == itemID {
            return item, true
        }
    }
    return ServiceItem{}, false
}

// LatestPerformance 返回服务项最近一次履约事件。
func (c Case) LatestPerformance(itemID string) (PerformanceEvent, bool) {
    var latest PerformanceEvent
    found := false
    for _, perf := range c.Performance {
        if perf.ItemID != itemID {
            continue
        }
        if !found || perf.OccurredAt.After(latest.OccurredAt) {
            latest = perf
            found = true
        }
    }
    return latest, found
}

// LatestFX 返回指定币种对在 at 之前（含）最近的汇率时点。
func (c Case) LatestFX(base, quote string, at time.Time) (FXPoint, bool) {
    var latest FXPoint
    found := false
    for _, fx := range c.FX {
        if fx.Base != base || fx.Quote != quote || fx.AsOf.After(at) {
            continue
        }
        if !found || fx.AsOf.After(latest.AsOf) {
            latest = fx
            found = true
        }
    }
    return latest, found
}
