package com.mudgame.storage;

import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.Room;

public class RoomStorage {
    private final RoomRepository repository;
    public RoomStorage(RoomRepository repository){
        this.repository=repository;
    }
    public Room loadRoom(String roomId) throws LoadException{
        Room staticRoom=RoomRepository.getStaticRoom(roomId);
        if(staticRoom==null){
            throw new LoadException("加载静态房间失败："+roomId);
        }

        Room dynamicRoom;
        try{
            dynamicRoom=repository.loadDynamicRoom(roomId);
        }catch(Exception e){
            throw new LoadException("加载动态房间失败："+roomId,e);
        }

        Room room=staticRoom.clone();
        room.getPlayersInRoom().addAll(dynamicRoom.getPlayersInRoom());
        
        if (!dynamicRoom.getNpcs().isEmpty()) {
             room.setNpcs(dynamicRoom.getNpcs());
        }

        return room;
    }

    public void saveRoom(Room room) throws SaveException {
        try {
            repository.saveDynamicRoom(room);
        } catch (Exception e) {
            throw new SaveException("存储动态房间失败："+room.getId(),e);
        }
    }
}
