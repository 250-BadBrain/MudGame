package com.mudgame.exception;

public class LoadException extends Exception {
    public LoadException(String message) {
        super(message);
    }

    public LoadException(String message, Throwable cause) {
        super (message,cause);
    }
}
