package com.mudgame.protocal;

import java.util.List;
import java.util.Map;

import com.mudgame.utils.Logger;

public class GameResult {
    private String type; // "command"/"chat"/"system"
    private String subtype; // 子类型
    private boolean flag; // 提示指令是否正确执行
    private Long ts;// 时间戳
    private Map<String, Object> results; // 处理结果
    private List<String> logs; // 信息，包含所有信息，可选

    public GameResult() {

    }

    public GameResult(String type, String subType, boolean flag, Long ts, Map<String, Object> results, List<String> logs) {
        this.type = type;
        this.subtype = subType;
        this.flag = flag;
        this.ts = ts;
        this.results = results;
        this.logs = logs;
    }

    public String getType() {
        return type;
    }

    public String getSubtype() {
        return subtype;
    }

    public boolean getFlag() {
        return flag;
    }

    public Long getTs() {
        return ts;
    }

    public Map<String, Object> getResults() {
        return results;
    }

    public List<String> getLogs() {
        return logs;
    }

    public void printLogs() {
        if (logs == null) {
            return;
        }
        for (String log : logs) {
            Logger.info(log);
        }
    }


}
