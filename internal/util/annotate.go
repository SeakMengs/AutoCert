package util

import "github.com/SeakMengs/AutoCert/internal/constant"

func GetSignatureStatus(status constant.SignatoryStatus) string {
	switch status {
	case constant.SignatoryStatusNotInvited:
		return "Not Invited"
	case constant.SignatoryStatusInvited:
		return "Invited"
	case constant.SignatoryStatusSigned:
		return "Signed"
	case constant.SignatoryStatusRejected:
		return "Rejected"
	default:
		return "Unknown Status"
	}
}
