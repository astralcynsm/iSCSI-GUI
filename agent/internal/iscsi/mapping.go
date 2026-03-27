package iscsi

func ValidLunID(lunID int) bool {
	return lunID >= 0 && lunID <= 65535
}
