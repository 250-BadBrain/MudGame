package com.mudgame.exception;


public class UnknownCommandException extends RuntimeException {
    public UnknownCommandException(String Type) {
        super("Unknown command type: " + Type);
    }

    public UnknownCommandException(String Type, String SubType) {
        super("Unknown command of " + Type + ": " + SubType);
    }

    public UnknownCommandException(String Type, String SubType, String content) {
        super("Invalid content of " + Type + "-" + SubType + ": " + content);
    }
}
