package engine

func BuildPublicChat(sender, message string) map[string]interface{} {
	return map[string]interface{}{"senderName": sender, "message": message}
}
