package com.mudgame.engine.handler;

import com.mudgame.exception.UnknownCommandException;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

public class UnknownCommandHandler implements CommandHandler {
    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        String type = message.getType();
        String subType = message.getSubtype();
        if (type.equals("system") || type.equals("command") || type.equals("chat")) {
            throw new UnknownCommandException(type, subType);
        } else {
            throw new UnknownCommandException(type);
        }

    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }
}//未知的命令处理
