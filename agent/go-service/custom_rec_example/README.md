# custom_rec_example

这个示例演示了：在 **同一张截图**（`arg.Img`）上，在 Go `CustomRecognition` 里多次调用 `ctx.RunRecognition(...)`。

示例流程：

1. 对同一张 `arg.Img` 做一次 `TemplateMatch`，使用 `DailyRewards/EventRedDot.png`。
2. 继续对同一张 `arg.Img` 再做一次 `TemplateMatch`，使用 `Resell/back.png`。
3. 针对第一步命中的前几个模板框，继续在同一张 `arg.Img` 上做 OCR。
4. 把模板命中框、OCR ROI、OCR 文本统一放进 `CustomRecognitionResult.Detail` 的 JSON 里返回。

## 注册名

```text
CustomRecExampleMultiRecognition
```

## 最小 Pipeline 用法

可直接参考同目录下的 `pipeline_example.jsonc`。

## 说明

- 这个示例为了自包含，内部通过 `ctx.RunRecognition(nodeName, img, config)` 动态构造了 TemplateMatch / OCR 节点配置，所以不依赖额外的 pipeline 识别节点。
- Template 图片路径使用的是仓库里现成资源，主要目的是演示“同一张图多次识别”的写法；实际业务里请替换成你自己的模板、ROI 和 OCR 规则。
- 坐标与 ROI 仍以 **1280×720** 为基准。
