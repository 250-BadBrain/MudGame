package com.mudgame.config;

import com.mudgame.network.MudWebSocketHandler;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.socket.config.annotation.EnableWebSocket;
import org.springframework.web.socket.config.annotation.WebSocketConfigurer;
import org.springframework.web.socket.config.annotation.WebSocketHandlerRegistry;

@Configuration
@EnableWebSocket
public class WebSocketConfig implements WebSocketConfigurer {

    private final MudWebSocketHandler handler;
    private final String[] allowedOriginPatterns;

    public WebSocketConfig(
        MudWebSocketHandler handler,
        @Value("${mud.websocket.allowed-origin-patterns:http://localhost:*,http://127.0.0.1:*,http://[::1]:*,http://192.168.*:*,http://10.*:*,https://localhost:*,https://127.0.0.1:*,https://*.ngrok-free.app,https://*.ngrok.io}")
        String allowedOriginPatterns) {
        this.handler = handler;
        this.allowedOriginPatterns = allowedOriginPatterns.split("\\s*,\\s*");
    }

    @SuppressWarnings("null")
    @Override
    public void registerWebSocketHandlers(WebSocketHandlerRegistry registry) {
        registry.addHandler(handler, "/ws")   // WS 访问路径
                .setAllowedOriginPatterns(allowedOriginPatterns);
    }
}
