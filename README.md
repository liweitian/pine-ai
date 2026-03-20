## Self Report
- 总耗时：6⼩时
- 实际做题时间段：16:30~ 22:30
- 完成情况：
- [x] 模型注册 / 更新 / 查看
- [x] 流式推理接⼝ （SSE足够，未看到需要双向传输数据的场景）
- [x] 热更新不影响已有连接
- [x] 多版本分流
- [x] Prometheus metrics
- [x] 简易管理⾯板 （apiV1.GET("/models", v1.ListModelsAPI) 查看）
- [x] 多后端模型动态切换 （根据backend_type，如果修改了可动态切换）
- [x] 模型热重启与资源回收机制 会记录上次使用时间，后台启动了脚本，每小时检查一次，如果超时则设置上deleted
- 备注说明：
- 当某个推理开始时，会创建当前任务快照，有唯一id,后续如需保持上下文一致，可在任务结束前拿到唯一id，可查看任务所用模型+版本
- 未做过多参数校验，后面给了test case，按照test case，可基本覆盖完成的场景

项目构建于golang 1.26 gin框架
1. go mod tidy 下载依赖
2. make build 构建
3. make run 运行web服务器于8080端口
可提供运行时的环境变量用于创建openai client端, 或直接改文件中的api-key，查看文件client/llm/open-ai.go

//持久层定义如下
type ModelRecord struct {
	ModelName     string           `json:"model_name"`
	Version       string           `json:"version"`
	BackendType   enum.BackendType `json:"backend_type"`
	Concurrency   int              `json:"concurrency"`
	Weight        int              `json:"weight"`
	Deleted       bool             `json:"deleted"`
	State         State            `json:"state"`
	LastUsedAt    time.Time        `json:"last_used_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

backend_type为 "openai" | "ollama" | "qwen" |  "mock"
其中如果提供openai的api-key则可以调用openai的接口，否则会报错，启动时需要添加环境变量。其它backend_type实际为本地mock，
version可用于处理实际的大模型版本 比如backend_type为 "openai", version可以是gpt3.5

在命令行中依次运行以下命令
BASE="http://127.0.0.1:8080/api/v1/pine-ai"

CASE 1:
创建一个mock模型，设置并发度为2
curl -i -X POST "$BASE/models" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "chat-bot",
    "version": "v1",
    "backend_type": "mock",
    "concurrency": 2,
    "weight": 100
  }'

开始推理
curl -N -i -X POST "$BASE/infer" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "chat-bot",
    "version": "v1",
    "input": "hello"
  }'

在另一命令行中再次输入以上命令，可看到同时流式返回

CASE 2:
更新已有模型，修改并发度为1
curl -i -X PUT "$BASE/models/chat-bot/version/v1" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "chat-bot",
    "version": "v1",
    "backend_type": "mock",
    "concurrency": 1,
    "weight": 100
  }'

开始推理
curl -N -i -X POST "$BASE/infer" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "chat-bot",
    "version": "v1",
    "input": "hello"
  }'

在另一命令行中再次输入以上命令，可看到报错信息

CASE 3
curl -N -i -X POST "$BASE/infer" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "chat-bot",
    "version": "v1",
    "input": "hello"
  }'
推理过程中调用下面的接口可以看到实时状态，比如可以看到in_use_count为1，表明当前模型正在使用
curl -i "http://127.0.0.1:8080/api/v1/pine-ai/models"

CASE 4
curl -i -X POST "$BASE/models" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "chat-bot",
    "version": "gpt3.5",
    "backend_type": "openai",
    "upstream_model": "mock-default",
    "concurrency": 2,
    "weight": 100
  }'

curl -N -i -X POST "$BASE/infer" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "chat-bot",
    "version": "gpt3.5",
    "input": "hello"
  }'

为避免传参出错，openai接口使用的模型固定为openai.GPT4o，
开发时使用的是Azure的openai服务，非直连openai,可能会有出入

CASE 5:
创建一个mock模型 版本为v1
curl -i -X POST "$BASE/models" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "chat-bot",
    "version": "v1",
    "backend_type": "mock",
    "concurrency": 1,
    "weight": 50
  }'

创建一个mock模型 版本为v2
curl -i -X POST "$BASE/models" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "chat-bot",
    "version": "v2",
    "backend_type": "mock",
    "concurrency": 1,
    "weight": 50
  }'

不指定具体版本时，按照创建时的权重随机选择版本
curl -N -i -X POST "$BASE/infer" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "chat-bot",
    "input": "hello"
  }'