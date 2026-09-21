package tourismdisputeresolution

import (
    "os"
    "testing"
)

func TestFixtureMatchesDomain(t *testing.T) {
    raw, err := os.ReadFile("../fixtures/domain.json")
    if err != nil { t.Fatal(err) }
    value, err := Parse(raw)
    if err != nil { t.Fatal(err) }
    if value.Domain != "tourism-dispute-resolution" { t.Fatalf("领域标识不一致: %s", value.Domain) }
    if len(value.Workflow) < 2 { t.Fatal("流程阶段缺失") }
    if value.Workflow[0] != "争议登记" || value.Workflow[len(value.Workflow)-1] != "执行核对与结案" {
        t.Fatalf("流程顺序异常: %v", value.Workflow)
    }
}
