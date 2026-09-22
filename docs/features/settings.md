# 系统设置

系统设置页集中管理面板级配置，包含以下分组。

## SSL 证书

- 支持禁用、Let's Encrypt、自签名、上传证书四种模式。
- 配置目标域名和邮箱后可自动申请/续期 Let's Encrypt 证书。
- 保存后可以立即重启服务生效，或稍后手动重启 `systemd` 服务。

## 面板访问来源

- 可开启面板访问白名单，仅允许指定 IP/网段访问 Web 面板。
- 可配置可信反向代理地址，避免代理环境下误拦截。
- 与 `clicd access-policy` CLI 等效，保存后立即生效。

## 通知推送

- 安全告警支持 Webhook 推送。
- 支持 SMTP 邮件推送，可配置服务器、端口、账号和收件人。

## 任务队列

- 配置任务队列并发数，控制同时执行的重装/创建/镜像任务数量。
- 保存后立即生效。

## 账号

- 修改管理员用户名和密码。
- 修改时需要输入当前密码确认。

## 其他

- WebSSH/WebVNC Origin 白名单。
- 语言设置（简体中文 / English）。

## 相关接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/PUT | `/api/ssl` | SSL 设置 |
| GET/PUT | `/api/access-policy` | 面板访问来源策略 |
| GET/PUT | `/api/notifications` | 通知推送设置 |
| GET/PUT | `/api/task-queue/settings` | 任务队列并发 |
| POST | `/api/change-password` | 修改密码 |
| POST | `/api/change-username` | 修改用户名 |
| GET/PUT | `/api/webssh-origins` | WebSSH/WebVNC Origin 白名单 |
