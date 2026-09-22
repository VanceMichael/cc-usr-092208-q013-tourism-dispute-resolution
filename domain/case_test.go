package tourismdisputeresolution

import (
    "os"
    "strings"
    "testing"
    "time"
)

func loadCase(t *testing.T) Case {
    t.Helper()
    raw, err := os.ReadFile("../fixtures/case.json")
    if err != nil {
        t.Fatal(err)
    }
    c, err := ParseCase(raw)
    if err != nil {
        t.Fatalf("示例案件应当通过校验: %v", err)
    }
    return c
}

func TestCaseFixtureLoads(t *testing.T) {
    c := loadCase(t)
    if c.ID != "AJ-2026-013" {
        t.Fatalf("案件标识不一致: %s", c.ID)
    }
    if len(c.Contract.Items) != 3 || len(c.Claims) != 4 || len(c.Events) != 11 {
        t.Fatalf("案件资料规模异常: 服务项 %d, 索赔 %d, 事件 %d",
            len(c.Contract.Items), len(c.Claims), len(c.Events))
    }
}

func TestFindDuplicateClaimsFlagsButDoesNotDecide(t *testing.T) {
    c := loadCase(t)
    found := FindDuplicateClaims(c.Claims)
    if len(found) != 1 {
        t.Fatalf("应发现 1 对疑似重复索赔，实际 %d 对", len(found))
    }
    pair := found[0]
    if pair.FirstClaimID != "clm-001" || pair.SecondClaimID != "clm-002" {
        t.Fatalf("疑似重复对应为 clm-001 与 clm-002，实际 %s 与 %s",
            pair.FirstClaimID, pair.SecondClaimID)
    }
    if pair.Reason == "" {
        t.Fatal("疑似重复必须附带理由，供人工裁决参考")
    }
    // 系统只提示，不改动任何索赔状态。
    for _, claim := range c.Claims {
        if claim.ID == "clm-002" && claim.Status != ClaimOpen {
            t.Fatalf("重复索赔识别不得替代人工裁决，clm-002 状态被改动为 %s", claim.Status)
        }
    }
}

func TestPendingViewShowsUnresolvedAndNextParty(t *testing.T) {
    c := loadCase(t)

    jia := PendingView(c, "游客甲")
    if len(jia) != 2 {
        t.Fatalf("游客甲应有 2 个未决项目，实际 %d", len(jia))
    }
    for _, item := range jia {
        if item.ItemID != "svc-flight" || item.NextParty != "迅航承运" {
            t.Fatalf("游客甲未决项目异常: %+v", item)
        }
    }

    yi := PendingView(c, "游客乙")
    if len(yi) != 1 {
        t.Fatalf("游客乙应有 1 个未决项目（已和解的导游索赔不应出现），实际 %d", len(yi))
    }
    // next_party 未记录时回退为履约企业。
    if yi[0].ItemID != "svc-scenic" || yi[0].NextParty != "云山景区" {
        t.Fatalf("游客乙未决项目异常: %+v", yi[0])
    }
}

func TestMaterialsForScopesEnterpriseAccess(t *testing.T) {
    c := loadCase(t)

    if got := len(MaterialsFor(c, RoleMediator, "调解中心")); got != 5 {
        t.Fatalf("调解人员应见全部 5 份材料，实际 %d", got)
    }
    if got := len(MaterialsFor(c, RoleTourist, "游客甲")); got != 5 {
        t.Fatalf("案件游客应见全部 5 份材料，实际 %d", got)
    }

    scenic := MaterialsFor(c, RoleSupplier, "云山景区")
    if len(scenic) != 2 {
        t.Fatalf("景区只能访问与自身服务有关的 2 份材料，实际 %d", len(scenic))
    }
    for _, ev := range scenic {
        if ev.ItemID != "svc-scenic" {
            t.Fatalf("景区看到了无关材料: %s", ev.ID)
        }
    }

    carrier := MaterialsFor(c, RoleSupplier, "迅航承运")
    if len(carrier) != 2 {
        t.Fatalf("承运人只能访问与自身服务有关的 2 份材料，实际 %d", len(carrier))
    }

    operator := MaterialsFor(c, RoleOperator, "韩进旅行社")
    if len(operator) != 4 {
        t.Fatalf("组团社可见合同内 4 份服务项材料，实际 %d", len(operator))
    }

    payment := MaterialsFor(c, RolePayment, "跨境付")
    if len(payment) != 2 {
        t.Fatalf("支付机构只访问票据类材料，应得 2 份，实际 %d", len(payment))
    }
    for _, ev := range payment {
        if ev.Kind != EvidenceReceipt {
            t.Fatalf("支付机构看到了非票据材料: %s", ev.ID)
        }
    }
}

func TestProposeSettlementIsItemizedAndExplainable(t *testing.T) {
    c := loadCase(t)
    at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
    settlement, err := ProposeSettlement(c, "KRW", at)
    if err != nil {
        t.Fatal(err)
    }
    // 已和解的 clm-004 与已履约的 svc-guide 不产生条目。
    if len(settlement.Items) != 3 {
        t.Fatalf("应生成 3 个和解条目，实际 %d", len(settlement.Items))
    }

    byClaim := make(map[string]SettlementItem)
    for _, item := range settlement.Items {
        byClaim[item.ClaimID] = item
        if item.Explanation == "" || len(item.Basis) < 2 {
            t.Fatalf("和解条目 %s/%s 缺少逐项解释或依据", item.ClaimID, item.ItemID)
        }
    }

    flight := byClaim["clm-001"]
    if flight.Refund.Currency != "KRW" || flight.Refund.Amount != 228000 {
        t.Fatalf("境内段取消应全额退款 1200 CNY × 190 = 228000 KRW，实际 %+v", flight.Refund)
    }
    if flight.Responsible != "迅航承运" {
        t.Fatalf("承运人已自认责任，责任方应为迅航承运，实际 %s", flight.Responsible)
    }
    if !hasBasisContaining(flight, "汇率时点") {
        t.Fatal("跨币种条目必须引用汇率时点作为依据")
    }

    scenic := byClaim["clm-003"]
    if scenic.Refund.Amount != 45600 {
        t.Fatalf("景区半日履约应按 50%% 计赔 240 CNY × 190 = 45600 KRW，实际 %+v", scenic.Refund)
    }
    if scenic.Responsible != "华东地接社" {
        t.Fatalf("地接社自认衔接责任，责任方应为华东地接社，实际 %s", scenic.Responsible)
    }
}

func TestProposeSettlementRequiresFXPoint(t *testing.T) {
    c := loadCase(t)
    c.FX = nil
    _, err := ProposeSettlement(c, "KRW", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
    if err == nil || !strings.Contains(err.Error(), "汇率时点") {
        t.Fatalf("缺少汇率时点时必须报错而不是猜测汇率，实际: %v", err)
    }
}

func TestVerifyExecutionFindsGaps(t *testing.T) {
    c := loadCase(t)
    now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
    gaps := VerifyExecution(c.Executions, now)

    byID := make(map[string]string)
    for _, gap := range gaps {
        byID[gap.ExecutionID] = gap.Problem
    }
    if _, ok := byID["exe-001"]; ok {
        t.Fatal("exe-001 按期足额执行，不应出现落差")
    }
    if byID["exe-002"] != "执行金额与约定不符" {
        t.Fatalf("exe-002 应为金额不符，实际: %s", byID["exe-002"])
    }
    if byID["exe-003"] != "执行逾期" {
        t.Fatalf("exe-003 应为执行逾期，实际: %s", byID["exe-003"])
    }
    if byID["exe-004"] != "超过期限仍未执行" {
        t.Fatalf("exe-004 应为超期未执行，实际: %s", byID["exe-004"])
    }
}

func TestValidateEventOrderRejectsBrokenSequence(t *testing.T) {
    base := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
    events := []CaseEvent{
        {Seq: 1, Kind: EventCaseOpened, Actor: "游客甲", OccurredAt: base},
        {Seq: 3, Kind: EventCompanionSplit, Actor: "调解中心", OccurredAt: base.Add(time.Hour)},
    }
    if err := ValidateEventOrder(events); err == nil {
        t.Fatal("事件编号跳号必须被拒绝")
    }

    events[1].Seq = 2
    events[1].OccurredAt = base.Add(-time.Hour)
    if err := ValidateEventOrder(events); err == nil {
        t.Fatal("发生时间倒退必须被拒绝")
    }
}

func TestValidateRejectsUnknownItemReference(t *testing.T) {
    c := loadCase(t)
    c.Claims[0].ItemIDs = []string{"svc-missing"}
    if err := c.Validate(); err == nil {
        t.Fatal("索赔引用未知服务项必须被拒绝")
    }
}

func hasBasisContaining(item SettlementItem, keyword string) bool {
    for _, basis := range item.Basis {
        if strings.Contains(basis, keyword) {
            return true
        }
    }
    return false
}
