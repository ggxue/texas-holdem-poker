# 牌桌升级：审查前提交整理（2026-10-09）

用户要求在审查前整理提交名称和未提交内容。仓库未配置远程；此次仅整理本地历史。

- 最近连续8条无类型前缀的提交统一为`feat:`／`test:`，作者、提交时间和各提交的tree均保留；父引用随新SHA更新。
- 逐条验证原／新tree完全一致，原HEAD与整理后HEAD的文件差异为空；实现与测试代码未改。
- 当前票据、索引和验收证据中的baseline引用同步为等价新SHA。原RED／GREEN、双轴审查和浏览器结果未重新生成，仍描述相同实现内容。
- 收齐原先保留的讨论、规格、词汇、架构、交接和参考图；更新完成状态，历史讨论与拆票前交接快照继续保留。
- 原完整历史已保存到本地忽略文件`artifacts/commit-cleanup-before.bundle`并通过`git bundle verify`；备份未纳入版本库。

升级审查起点：`ed60ba67b62cb55ceacadfcf37a5824a6dd633a9`（等价于原实施起点）。
升级七票最终提交：`8f6759dfd64a961c7745e42e5c7732e0f42ef016`；其后文档整理提交见实时`git log`。

```powershell
git diff ed60ba67b62cb55ceacadfcf37a5824a6dd633a9...HEAD
```

| 原SHA | 整理后SHA | 提交名称 |
| --- | --- | --- |
| `bb30780165d285fb5f7529d6ac14f006cfc32851` | `ed60ba67b62cb55ceacadfcf37a5824a6dd633a9` | test: complete local poker acceptance and prepare free deployment |
| `569bb7461643dd110d2573af9f93238c08b3ec7a` | `4ff5d2e77b61d53997b08fdfed9880a35937930e` | feat: expand poker room to five humans and preserve host entry order |
| `33e1f13857b98e3e64b9ad7bf500105928f96c65` | `102f53b74906585f0610610ced97967818e514c3` | feat: randomize each hand action start and align odd-chip awards |
| `94e229eed678a0a2375218ec96e68d5bd7a2c4fc` | `cb67bfe3cb9c69d03ab43a7f30f5c08aedb9debe` | feat: delay bot actions with cancellable thinking deadlines |
| `13edbf10093dea3b22cdc2dcef6a4afe677ce2e4` | `2f5ec8a8fa92ae36ce60f9d764ae4a76cf247cf2` | feat: arrange fixed poker seats for desktop and phone |
| `82cdccd01df8aa77f9c354bf501f2f8e87bf001a` | `4eb7b665a8e3780ab7d034a528f934d2adf6f161` | feat: render simple cards and a complete hand reference |
| `af317c20ce24b04f8012de6d74fe403938e27c23` | `e5f61481d54cb9e85c7759fbb075eb0f8bb37a08` | feat: show confirmed settlement results below player actions |
| `9f2777fafddaae5808e13f77b6f963b6eeb6e4ee` | `8f6759dfd64a961c7745e42e5c7732e0f42ef016` | test: complete poker table upgrade acceptance and regression evidence |

文档整理检查：Harness、Git空白格式及本地Markdown链接通过。此次仅调整文档和提交元数据；原功能验证与外部限制见[验收证据](acceptance-evidence.md)。
