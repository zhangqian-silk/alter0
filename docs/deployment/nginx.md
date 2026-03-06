# Nginx 反向代理与登录保护

本方案用于公网部署场景：`alter0` 仅监听本机回环地址，由 Nginx 对外暴露，并通过应用内登录页进行访问控制。

## 1. 运行参数（最小安全基线）

```bash
export ALTER0_WEB_LOGIN_PASSWORD='请替换为强密码'

go run ./cmd/alter0 \
  -web-addr 127.0.0.1:18088 \
  -web-bind-localhost-only=true \
  -web-login-password "$ALTER0_WEB_LOGIN_PASSWORD"
```

说明：

1. `web-bind-localhost-only=true` 会强制服务只监听本机（避免直接公网暴露）。
2. `web-login-password` 非空时启用登录页；访问 `/chat` 需先登录。
3. `web-login-password` 为空时关闭登录页（不建议公网环境使用）。

## 2. Nginx 示例配置

参考 `docs/deployment/nginx.alter0.conf`，核心配置如下：

```nginx
server {
    listen 80;
    server_name your.domain.com;

    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:18088;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}
```

如果已启用 HTTPS，请保留 `X-Forwarded-Proto`，用于安全 Cookie 判定。

## 3. 最简单密码配置方案

推荐使用单独的环境文件，并收紧文件权限：

```bash
sudo install -d -m 750 /etc/alter0
sudo sh -c "printf 'ALTER0_WEB_LOGIN_PASSWORD=请替换为强密码\n' > /etc/alter0/alter0.env"
sudo chmod 600 /etc/alter0/alter0.env
```

在 systemd 中加载：

```ini
EnvironmentFile=/etc/alter0/alter0.env
```

该方案简单直接，避免把密码写进命令历史。

## 4. 运行权限建议

1. 使用专用低权限用户运行（例如 `alter0`）。
2. 运行目录建议权限：目录 `750`、日志 `640`、密码环境文件 `600`。
3. 避免直接使用 root 长期运行服务。

项目内提供了最小化启动脚本：`scripts/start_alter0_service.sh`。
