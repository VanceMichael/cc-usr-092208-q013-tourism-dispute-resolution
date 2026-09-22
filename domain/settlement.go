package tourismdisputeresolution

import (
    "fmt"
    "sort"
    "time"
)

// SettlementItem 表示和解方案中一个逐项可解释的条目。
type SettlementItem struct {
    ClaimID     string   `json:"claim_id"`
    ItemID      string   `json:"item_id"`
    Responsible string   `json:"responsible"` // 责任方
    Refund      Money    `json:"refund"`      // 折算为游客收单币种后的退款
    Basis       []string `json:"basis"`       // 依据：合同、履约事件、证据、汇率时点
    Explanation string   `json:"explanation"` // 逐项说明
}

// Settlement 表示调解人员生成的和解方案，不自动生效，需各方确认。
type Settlement struct {
    CaseID string           `json:"case_id"`
    Items  []SettlementItem `json:"items"`
}

// ProposeSettlement 以原合同和实际履约事件为基础，为未决索赔逐项生成
// 可解释的和解方案：退款按合同价与履约缺口比例计算，再以最近汇率时点
// 折算为游客收单币种；责任方取自认责任的一方，否则为履约企业。
func ProposeSettlement(c Case, payoutCurrency string, at time.Time) (Settlement, error) {
    if payoutCurrency == "" {
        return Settlement{}, fmt.Errorf("缺少收单币种")
    }
    result := Settlement{CaseID: c.ID}
    for _, claim := range c.Claims {
        if claimResolved(claim.Status) {
            continue
        }
        for _, itemID := range claim.ItemIDs {
            item, ok := c.Item(itemID)
            if !ok {
                return Settlement{}, fmt.Errorf("索赔 %s 引用了未知服务项: %s", claim.ID, itemID)
            }
            entry, ok, err := c.settleItem(claim, item, payoutCurrency, at)
            if err != nil {
                return Settlement{}, err
            }
            if ok {
                result.Items = append(result.Items, entry)
            }
        }
    }
    sort.Slice(result.Items, func(i, j int) bool {
        if result.Items[i].ClaimID != result.Items[j].ClaimID {
            return result.Items[i].ClaimID < result.Items[j].ClaimID
        }
        return result.Items[i].ItemID < result.Items[j].ItemID
    })
    return result, nil
}

// settleItem 计算单个索赔服务项的和解条目；已完全履约的项目不产生退款。
func (c Case) settleItem(claim Claim, item ServiceItem, payoutCurrency string, at time.Time) (SettlementItem, bool, error) {
    perf, ok := c.LatestPerformance(item.ID)
    if !ok {
        return SettlementItem{}, false, fmt.Errorf("服务项 %s 缺少履约事件，无法归并", item.ID)
    }
    if perf.Status == PerformanceFulfilled {
        return SettlementItem{}, false, nil
    }
    shortfall := perf.Shortfall
    if perf.Status == PerformanceCancelled {
        shortfall = 1
    }
    if shortfall <= 0 {
        return SettlementItem{}, false, nil
    }

    refund := Money{Amount: item.Price.Amount * shortfall, Currency: item.Price.Currency}
    basis := []string{
        fmt.Sprintf("合同 %s 服务项 %s（%s，%0.2f %s）", c.Contract.ID, item.ID, item.Title, item.Price.Amount, item.Price.Currency),
        fmt.Sprintf("履约事件：%s，缺口比例 %0.2f（%s）", perf.Status, shortfall, perf.Note),
    }
    for _, ev := range c.Evidence {
        if ev.ItemID == item.ID {
            basis = append(basis, fmt.Sprintf("证据 %s（%s）", ev.ID, ev.Kind))
        }
    }
    if refund.Currency != payoutCurrency {
        fx, ok := c.LatestFX(refund.Currency, payoutCurrency, at)
        if !ok {
            return SettlementItem{}, false, fmt.Errorf("缺少 %s 兑 %s 的汇率时点", refund.Currency, payoutCurrency)
        }
        refund = Money{Amount: refund.Amount * fx.Rate, Currency: payoutCurrency}
        basis = append(basis, fmt.Sprintf("汇率时点：%s/%s=%g（%s）", fx.Base, fx.Quote, fx.Rate, fx.AsOf.Format("2006-01-02")))
    }

    responsible := item.Supplier
    for _, op := range c.Opinions {
        if op.ItemID == item.ID && op.Accepts {
            responsible = op.Party
        }
    }
    explanation := fmt.Sprintf(
        "服务项「%s」%s，按合同价 %0.2f %s 的 %0.0f%% 计赔，折算为 %0.2f %s，由 %s 承担。",
        item.Title, perf.Status, item.Price.Amount, item.Price.Currency,
        shortfall*100, refund.Amount, refund.Currency, responsible,
    )
    return SettlementItem{
        ClaimID:     claim.ID,
        ItemID:      item.ID,
        Responsible: responsible,
        Refund:      refund,
        Basis:       basis,
        Explanation: explanation,
    }, true, nil
}
