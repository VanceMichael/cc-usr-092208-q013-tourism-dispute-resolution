package tourismdisputeresolution

import "fmt"

// DuplicateCandidate 表示一对疑似重复的索赔。
// 系统只负责发现并提示，是否合并或驳回始终由人工裁决。
type DuplicateCandidate struct {
    FirstClaimID  string `json:"first_claim_id"`
    SecondClaimID string `json:"second_claim_id"`
    Reason        string `json:"reason"`
}

// FindDuplicateClaims 自动发现疑似重复索赔：
// 同一索赔人就同一服务项再次索赔，或以相同事由、相同金额重复提交。
// 结果仅供调解人员复核，不替代人工裁决。
func FindDuplicateClaims(claims []Claim) []DuplicateCandidate {
    var found []DuplicateCandidate
    for i := 0; i < len(claims); i++ {
        for j := i + 1; j < len(claims); j++ {
            a, b := claims[i], claims[j]
            if a.Claimant != b.Claimant {
                continue
            }
            if shared := sharedItems(a.ItemIDs, b.ItemIDs); len(shared) > 0 {
                found = append(found, DuplicateCandidate{
                    FirstClaimID:  a.ID,
                    SecondClaimID: b.ID,
                    Reason:        fmt.Sprintf("同一索赔人就服务项 %v 重复索赔", shared),
                })
                continue
            }
            if a.Grounds == b.Grounds && a.Amount == b.Amount {
                found = append(found, DuplicateCandidate{
                    FirstClaimID:  a.ID,
                    SecondClaimID: b.ID,
                    Reason:        "同一索赔人以相同事由和金额重复提交",
                })
            }
        }
    }
    return found
}

func sharedItems(a, b []string) []string {
    set := make(map[string]bool, len(b))
    for _, id := range b {
        set[id] = true
    }
    var shared []string
    for _, id := range a {
        if set[id] {
            shared = append(shared, id)
        }
    }
    return shared
}
