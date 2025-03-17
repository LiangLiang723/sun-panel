package system

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sun-panel/api/api_v1/common/apiData/commonApiStructs"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cmn"
	"sun-panel/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

type FileApi struct{}

func (a *FileApi) UploadImg(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	configUpload := global.Config.GetValueString("base", "source_path")
	f, err := c.FormFile("imgfile")
	if err != nil {
		apiReturn.ErrorByCode(c, 1300)
		return
	} else {
		fileExt := strings.ToLower(path.Ext(f.Filename))
		agreeExts := []string{
			".png",
			".jpg",
			".gif",
			".jpeg",
			".webp",
			".svg",
			".ico",
		}

		if !cmn.InArray(agreeExts, fileExt) {
			apiReturn.ErrorByCode(c, 1301)
			return
		}
		fileName := cmn.Md5(fmt.Sprintf("%s%s", f.Filename, time.Now().String()))
		fildDir := fmt.Sprintf("%s/%d/%d/%d/", configUpload, time.Now().Year(), time.Now().Month(), time.Now().Day())
		isExist, _ := cmn.PathExists(fildDir)
		if !isExist {
			os.MkdirAll(fildDir, os.ModePerm)
		}
		filepath := fmt.Sprintf("%s%s%s", fildDir, fileName, fileExt)
		c.SaveUploadedFile(f, filepath)

		// 像数据库添加记录
		mFile := models.File{}
		mFile.AddFile(userInfo.ID, f.Filename, fileExt, filepath)
		apiReturn.SuccessData(c, gin.H{
			"imageUrl": filepath[1:],
		})
	}
}

func (a *FileApi) UploadFiles(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	configUpload := global.Config.GetValueString("base", "source_path")

	form, err := c.MultipartForm()
	if err != nil {
		apiReturn.ErrorByCode(c, 1300)
		return
	}
	files := form.File["files[]"]
	errFiles := []string{}
	succMap := map[string]string{}
	for _, f := range files {
		fileExt := strings.ToLower(path.Ext(f.Filename))
		fileName := cmn.Md5(fmt.Sprintf("%s%s", f.Filename, time.Now().String()))
		fildDir := fmt.Sprintf("%s/%d/%d/%d/", configUpload, time.Now().Year(), time.Now().Month(), time.Now().Day())
		isExist, _ := cmn.PathExists(fildDir)
		if !isExist {
			os.MkdirAll(fildDir, os.ModePerm)
		}
		filepath := fmt.Sprintf("%s%s%s", fildDir, fileName, fileExt)
		if c.SaveUploadedFile(f, filepath) != nil {
			errFiles = append(errFiles, f.Filename)
		} else {
			// 成功
			// 像数据库添加记录
			mFile := models.File{}
			mFile.AddFile(userInfo.ID, f.Filename, fileExt, filepath)
			succMap[f.Filename] = filepath[1:]
		}
	}

	apiReturn.SuccessData(c, gin.H{
		"succMap":  succMap,
		"errFiles": errFiles,
	})
}

func (a *FileApi) GetList(c *gin.Context) {
	type Request struct {
		Group string `json:"group"` // 可选的分组参数: all, original, renamed
	}

	req := Request{}
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		// 如果绑定失败，使用默认值"all"
		req.Group = "all"
	}

	list := []models.File{}
	userInfo, _ := base.GetCurrentUserInfo(c)
	var count int64

	db := global.Db.Where("user_id=?", userInfo.ID)

	// 根据分组过滤
	if req.Group == "renamed" {
		db = db.Where("src LIKE ?", "%/managed_user%")
	} else if req.Group == "original" {
		db = db.Where("src NOT LIKE ?", "%/managed_user%")
	}

	if err := db.Order("created_at desc").Find(&list).Count(&count).Error; err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	data := []map[string]interface{}{}
	for _, v := range list {
		fileType := "original"
		if strings.Contains(v.Src, "/managed_user") {
			fileType = "renamed"
		}

		data = append(data, map[string]interface{}{
			"src":        v.Src[1:],
			"fileName":   v.FileName,
			"id":         v.ID,
			"createTime": v.CreatedAt,
			"updateTime": v.UpdatedAt,
			"path":       v.Src,
			"fileType":   fileType,
		})
	}
	apiReturn.SuccessListData(c, data, count)
}

func (a *FileApi) Deletes(c *gin.Context) {
	req := commonApiStructs.RequestDeleteIds[uint]{}
	userInfo, _ := base.GetCurrentUserInfo(c)
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiReturn.ErrorParamFomat(c, err.Error())
		return
	}

	global.Db.Transaction(func(tx *gorm.DB) error {
		files := []models.File{}

		if err := tx.Order("created_at desc").Find(&files, "user_id=? AND id in ?", userInfo.ID, req.Ids).Error; err != nil {
			return err
		}

		for _, v := range files {
			os.Remove(v.Src)
		}

		if err := tx.Order("created_at desc").Delete(&files, "user_id=? AND id in ?", userInfo.ID, req.Ids).Error; err != nil {
			return err
		}

		return nil
	})

	apiReturn.Success(c)

}

// Rename 重命名文件
func (a *FileApi) Rename(c *gin.Context) {
	type Request struct {
		ID       uint   `json:"id"`
		FileName string `json:"fileName"`
		Force    bool   `json:"force"` // 是否强制覆盖现有文件
	}

	req := Request{}
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiReturn.ErrorParamFomat(c, err.Error())
		return
	}

	// 获取当前用户信息
	userInfo, _ := base.GetCurrentUserInfo(c)

	// 查找文件记录
	fileInfo := models.File{}
	if err := global.Db.First(&fileInfo, "id = ? AND user_id = ?", req.ID, userInfo.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apiReturn.ErrorDataNotFound(c)
			return
		} else {
			apiReturn.ErrorDatabase(c, err.Error())
			return
		}
	}

	// 文件系统操作 - 重命名并移动文件
	configUpload := global.Config.GetValueString("base", "source_path")

	// 创建统一管理的文件夹
	managedDir := fmt.Sprintf("%s/managed_user%d/", configUpload, userInfo.ID)
	isExist, _ := cmn.PathExists(managedDir)
	if !isExist {
		if err := os.MkdirAll(managedDir, os.ModePerm); err != nil {
			apiReturn.Error(c, fmt.Sprintf("Failed to create directory: %s", err.Error()))
			return
		}
	}

	// 获取原文件扩展名
	fileExt := path.Ext(fileInfo.Src)

	// 构建新文件名和路径
	newFileName := req.FileName
	if path.Ext(newFileName) == "" {
		newFileName = fmt.Sprintf("%s%s", newFileName, fileExt) // 如果新文件名没有扩展名，添加原扩展名
	}

	// 在managed目录中的新路径
	newFilePath := fmt.Sprintf("%s%s", managedDir, newFileName)

	// 检查目标文件是否已存在
	targetExists, _ := cmn.PathExists(newFilePath)
	if targetExists && !req.Force {
		// 如果目标文件已存在，且不是强制覆盖模式，返回冲突状态
		apiReturn.SuccessData(c, gin.H{
			"conflict":   true,
			"message":    "File with this name already exists",
			"targetPath": newFilePath,
		})
		return
	}

	// 移动并重命名文件
	if err := os.Rename(fileInfo.Src, newFilePath); err != nil {
		apiReturn.Error(c, fmt.Sprintf("Failed to rename file: %s", err.Error()))
		return
	}

	// 更新数据库记录
	updates := map[string]interface{}{
		"file_name": req.FileName,
		"src":       newFilePath,
	}

	if err := global.Db.Model(&fileInfo).Updates(updates).Error; err != nil {
		// 如果数据库更新失败，尝试将文件移回原位置
		os.Rename(newFilePath, fileInfo.Src)
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	apiReturn.Success(c)
}
