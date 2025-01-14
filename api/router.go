package api

import (
	"github.com/gin-gonic/gin"
)

// ConfigChatRouter configure gin router for chat (without gzip)
func ConfigChatRouter(router *gin.RouterGroup) {
	configGinChatRouter(router)
}

//ConfigGinRouter for non ws endpoints
func ConfigGinRouter(router *gin.Engine) {
	configGinStreamRestRouter(router)
	configGinUsersRouter(router)
	configGinCourseRouter(router)
	configGinDownloadRouter(router)
	configGinLectureHallApiRouter(router)
	configGinSexyApiRouter(router)
	configProgressRouter(router)
	configServerNotificationsRoutes(router)
	configTokenRouter(router)
	configWorkerRouter(router)
	configNotificationsRouter(router)
}
