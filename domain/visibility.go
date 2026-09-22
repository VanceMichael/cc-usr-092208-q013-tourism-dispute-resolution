package tourismdisputeresolution

import "sort"

// PendingItem 表示游客可见的一个未决项目及其下一责任方。
type PendingItem struct {
    ItemID    string `json:"item_id"`
    Title     string `json:"title"`
    ClaimID   string `json:"claim_id"`
    Status    string `json:"status"`
    NextParty string `json:"next_party"` // 下一责任方
}

// PendingView 汇总某游客（含拆分后的同行人）名下所有未决项目，
// 使其始终能看见还有哪些服务没有解决、下一步由谁负责。
func PendingView(c Case, claimant string) []PendingItem {
    var pending []PendingItem
    for _, claim := range c.Claims {
        if claim.Claimant != claimant || claimResolved(claim.Status) {
            continue
        }
        for _, itemID := range claim.ItemIDs {
            item, ok := c.Item(itemID)
            if !ok {
                continue
            }
            pending = append(pending, PendingItem{
                ItemID:    itemID,
                Title:     item.Title,
                ClaimID:   claim.ID,
                Status:    string(claim.Status),
                NextParty: c.nextParty(item),
            })
        }
    }
    sort.Slice(pending, func(i, j int) bool {
        if pending[i].ItemID != pending[j].ItemID {
            return pending[i].ItemID < pending[j].ItemID
        }
        return pending[i].ClaimID < pending[j].ClaimID
    })
    return pending
}

func claimResolved(status ClaimStatus) bool {
    return status == ClaimSettled || status == ClaimWithdrawn
}

// nextParty 返回服务项的下一责任方：优先取案件记录，缺省时为履约企业。
func (c Case) nextParty(item ServiceItem) string {
    if party, ok := c.NextParty[item.ID]; ok && party != "" {
        return party
    }
    return item.Supplier
}

// MaterialsFor 按参与方角色过滤可访问的证据材料：
// 调解人员与案件游客可见全部材料；企业（组团社、服务商）只能访问
// 与自身服务项有关的材料；支付机构只访问票据类材料。
func MaterialsFor(c Case, role Role, party string) []Evidence {
    var visible []Evidence
    for _, ev := range c.Evidence {
        if canAccess(c, role, party, ev) {
            visible = append(visible, ev)
        }
    }
    return visible
}

func canAccess(c Case, role Role, party string, ev Evidence) bool {
    switch role {
    case RoleMediator, RoleTourist:
        return true
    case RolePayment:
        return ev.Kind == EvidenceReceipt
    case RoleSupplier:
        return servesItem(c, party, ev.ItemID)
    case RoleOperator:
        // 组团社可见自己操作合同内的服务项材料。
        return c.Contract.Operator == party && ev.ItemID != ""
    default:
        return false
    }
}

func servesItem(c Case, supplier, itemID string) bool {
    if itemID == "" {
        return false
    }
    item, ok := c.Item(itemID)
    return ok && item.Supplier == supplier
}
