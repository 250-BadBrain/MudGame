package com.mudgame.engine;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.handler.UnknownCommandHandler;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import org.springframework.stereotype.Component;

import java.util.EnumMap;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Component
public class HandleCommand {
    private final Map<String, CommandHandler> handlersSys = new HashMap<>();
    private final Map<String, CommandHandler> handlersCom = new HashMap<>();
    private final Map<String, CommandHandler> handlersChat = new HashMap<>();
    private final UnknownCommandHandler unknownCommandHandler;

    public HandleCommand(List<CommandHandler> handlers, UnknownCommandHandler unknownCommandHandler) {
        this.unknownCommandHandler = unknownCommandHandler;
        for (CommandHandler handler : handlers) {
            if (handler == unknownCommandHandler) {
                continue;
            }

            switch (handler.getType()) {
                case "system" -> handlersSys.put(handler.getSubType(), handler);
                case "command" -> handlersCom.put(handler.getSubType(), handler);
                case "chat" -> handlersChat.put(handler.getSubType(), handler);
                default -> {
                }
            }
        }
    }

    public GameResult handleCommand(CommandMessage command) throws UnknownCommandException {
        CommandHandler handler = switch (command.getType()) {
            case "system" -> handlersSys.getOrDefault(command.getSubtype(), unknownCommandHandler);
            case "command" -> handlersCom.getOrDefault(command.getSubtype(), unknownCommandHandler);
            case "chat" -> handlersChat.getOrDefault(command.getSubtype(), unknownCommandHandler);
            default -> unknownCommandHandler;
        };

        return handler.handle(command);
    }
}