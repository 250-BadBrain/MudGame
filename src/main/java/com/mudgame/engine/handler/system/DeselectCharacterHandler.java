package com.mudgame.engine.handler.system;

import com.mudgame.exception.UnknownCommandException;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

public class DeselectCharacterHandler extends SystemCommandHandler {
    private static final String subtype = "deselect_character";

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到deselect_character指令");
        // 逻辑很简单，只要收到这个指令，就认为成功，后续由 WebSocketHandler 处理解绑
        return new GameResult(type, subtype, true, System.currentTimeMillis(), null, null);
    }
    
    @Override
    public String getSubType() {
        return subtype;
    }
    
    @Override
    public String getType() {
        return type;
    }
}
