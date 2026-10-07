# 02: 清理 GoLand 示例的检查失败

Status: draft
Type: implementation
Blocked by: None
Spec: 待用户开始代码工作时确定处理方式。

## What to build

使项目初始 Go 入口通过统一验证。当前仍是原始示例程序，没有扑克业务行为。
用户可以选择保留并修正示例，或在首个实现任务中替换入口；本任务不替用户选择。

## Acceptance criteria

- [ ] Go 格式检查通过。
- [ ] 消除 fmt.Println 将 %s 当成普通文本而触发的 vet 诊断。
- [ ] scripts/verify.ps1 的 Go vet、测试和构建全部通过。
- [ ] 不关闭 vet、不添加忽略规则、不把已有失败报告为成功。

## Validation

初始化实际发现：gofmt 要求第一个 `//TIP` 注释变为 `// TIP`；
`main.go:13:2: fmt.Println call has possible Printf formatting directive %s`。
`go test ./...` 的默认 vet 同样触发该问题。`go build` 成功，目前没有测试文件。
