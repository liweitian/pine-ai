## Self Report
- 总耗时：______ ⼩时
- 实际做题时间段：____ ~ ____
- 完成情况：
- [x] 模型注册 / 更新 / 查看
- [x] 流式推理接⼝
- [x] 热更新不影响已有连接
- [ ] 多版本分流
- [ ] Prometheus metrics
- [ ] 灰度发布
- 备注说明：
- （如对模型状态隔离的实现⽅式说明 / 关键设计决策 / 哪块逻辑写得不满意 / 下⼀步想优化的
点）

项目构建于golang 1.26

make build 构建
make run 运行web服务器于8080端口
可提供运行时的环境变量用于创建openai client端, 可直接改文件中的api-key client/llm/open-ai.go

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
其中如果提供openai的api-key则可以调用openai的接口，否则会报错，启动时需要添加环境变量。其它实际为本地mock端口，
version 用于处理实际的大模型版本 比如backend_type为 "openai", version可以是gpt3.5

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
