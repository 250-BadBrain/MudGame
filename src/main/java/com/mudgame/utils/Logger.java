package com.mudgame.utils;

import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Date;
import java.text.SimpleDateFormat;

public class Logger{
    private static final String LOG_FILE;
    private static final SimpleDateFormat DATE_FORMAT=new SimpleDateFormat("yyyy-MM-dd HH:mm:ss");

    static{
        if (new java.io.File("server").exists() && new java.io.File("server").isDirectory()) {
            LOG_FILE = "server/logs/server.log";
        } else {
            LOG_FILE = "logs/server.log";
        }

        java.io.File dir=new java.io.File(LOG_FILE).getParentFile();
        if(!dir.exists())dir.mkdirs();
    }

    private static void writeLog(String level,String message){
        String timestamp=DATE_FORMAT.format(new Date());
        String logLine="[%s] [%s] %s".formatted(timestamp, level, message);

        // System.out.println(logLine);

        try(PrintWriter out=new PrintWriter(new OutputStreamWriter(new FileOutputStream(LOG_FILE,true), StandardCharsets.UTF_8))){
            out.println(logLine);
        }catch(IOException e){
            System.err.println("写入日志失败: "+e.getMessage());
        }
    }
    public static void info(String msg){
        writeLog("INFO",msg);
    }
    public static void warn(String msg){
        writeLog("WARN",msg);
    }
    public static void error(String msg){
        writeLog("ERROR",msg);
    }
    public static void debug(String msg){
        writeLog("DEBUG",msg);
    }
}