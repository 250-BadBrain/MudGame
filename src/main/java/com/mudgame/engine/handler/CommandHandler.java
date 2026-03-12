package com.mudgame.engine.handler;

import com.mudgame.exception.UnknownCommandException;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

public interface CommandHandler {
    String log_name = "engine: ";
    String type = "unknown";
    String subtype = "unknown";

    GameResult handle(CommandMessage message) throws UnknownCommandException;
    String getType();
    String getSubType();
}
