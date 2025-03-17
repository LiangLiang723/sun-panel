package system

import (
	"fmt"
	"os"
	"path"
	"strconv"
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

	// 设置一个错误收集器
	var deleteErrors []string

	err := global.Db.Transaction(func(tx *gorm.DB) error {
		files := []models.File{}

		if err := tx.Order("created_at desc").Find(&files, "user_id=? AND id in ?", userInfo.ID, req.Ids).Error; err != nil {
			return err
		}

		for _, v := range files {
			// 清理路径，确保格式正确
			srcPath := v.Src
			// 移除可能存在的路径前缀问题
			srcPath = strings.Replace(srcPath, "/./", "/", -1)

			// 尝试删除文件
			err := os.Remove(srcPath)
			if err != nil {
				// 如果删除失败，尝试替代路径
				altPath := strings.TrimPrefix(srcPath, "/")
				err = os.Remove(altPath)

				// 如果仍然失败，记录错误，但继续处理其他文件
				if err != nil {
					deleteErrors = append(deleteErrors, fmt.Sprintf("Failed to delete file %s: %s", v.FileName, err.Error()))
				}
			}
		}

		// 即使物理文件删除失败，仍然删除数据库记录
		if err := tx.Delete(&files, "user_id=? AND id in ?", userInfo.ID, req.Ids).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	// 如果有删除错误，返回警告信息，但仍标记为成功（因为数据库记录已删除）
	if len(deleteErrors) > 0 {
		apiReturn.SuccessData(c, gin.H{
			"warnings": deleteErrors,
			"message":  "Some files could not be physically removed but database records were deleted",
		})
		return
	}

	apiReturn.Success(c)
}

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

	// 清理源文件路径，确保格式正确
	srcPath := fileInfo.Src
	// 移除可能存在的路径前缀问题（如 /./）
	srcPath = strings.Replace(srcPath, "/./", "/", -1)

	// 检查源文件是否存在
	srcExists, _ := cmn.PathExists(srcPath)
	if !srcExists {
		// 尝试不同的路径格式（移除开头的斜杠）
		altSrcPath := strings.TrimPrefix(srcPath, "/")
		srcExists, _ = cmn.PathExists(altSrcPath)
		if srcExists {
			srcPath = altSrcPath // 使用替代路径
		} else {
			apiReturn.Error(c, fmt.Sprintf("Source file not found: %s", srcPath))
			return
		}
	}

	// 创建统一管理的文件夹（确保路径规范）
	configUpload = strings.TrimSuffix(configUpload, "/") // 移除末尾斜杠
	managedDir := fmt.Sprintf("%s/managed_user%d/", configUpload, userInfo.ID)

	// 确保目录存在
	isExist, _ := cmn.PathExists(managedDir)
	if !isExist {
		// 尝试创建完整路径
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

	// 在managed目录中的新路径（确保没有双斜杠）
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
	if err := os.Rename(srcPath, newFilePath); err != nil {
		// 如果重命名失败，记录详细错误信息
		apiReturn.Error(c, fmt.Sprintf("Failed to rename file from '%s' to '%s': %s",
			srcPath, newFilePath, err.Error()))
		return
	}

	// 更新数据库记录
	updates := map[string]interface{}{
		"file_name": req.FileName,
		"src":       fmt.Sprintf("/%s", strings.TrimPrefix(newFilePath, "/")), // 确保存储的路径格式一致
	}

	if err := global.Db.Model(&fileInfo).Updates(updates).Error; err != nil {
		// 如果数据库更新失败，尝试将文件移回原位置
		os.Rename(newFilePath, srcPath)
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	apiReturn.Success(c)
}

func (a *FileApi) RefreshFiles(c *gin.Context) {
	// 获取当前用户信息
	userInfo, _ := base.GetCurrentUserInfo(c)
	configUpload := global.Config.GetValueString("base", "source_path")

	// 简化路径处理，防止路径问题
	configUpload = strings.TrimSuffix(configUpload, "/")

	// 事务处理
	err := global.Db.Transaction(func(tx *gorm.DB) error {
		// 1. 先清空当前用户的文件记录
		if err := tx.Where("user_id = ?", userInfo.ID).Delete(&models.File{}).Error; err != nil {
			return err
		}

		// 2. 扫描managed目录
		managedDir := fmt.Sprintf("%s/managed_user%d", configUpload, userInfo.ID)
		managedDirExists, _ := cmn.PathExists(managedDir)

		if managedDirExists {
			files, err := os.ReadDir(managedDir)
			if err == nil {
				for _, file := range files {
					if file.IsDir() || file.Name() == ".gitkeep" {
						continue
					}

					fileExt := path.Ext(file.Name())
					filePath := fmt.Sprintf("/%s/managed_user%d/%s", configUpload, userInfo.ID, file.Name())

					// 创建文件记录
					fileRecord := models.File{
						UserId:   userInfo.ID,
						FileName: file.Name(),
						Src:      filePath,
						Ext:      fileExt,
					}

					if err := tx.Create(&fileRecord).Error; err != nil {
						return err
					}
				}
			}
		}

		// 3. 扫描所有年份的年/月/日目录结构
		rootDir := fmt.Sprintf("%s", configUpload)
		rootDirExists, _ := cmn.PathExists(rootDir)

		if rootDirExists {
			// 读取根目录下的所有文件夹（可能的年份目录）
			years, err := os.ReadDir(rootDir)
			if err != nil {
				return err
			}

			for _, yearDir := range years {
				// 只处理目录，且跳过managed_user目录和非年份目录
				if !yearDir.IsDir() || strings.HasPrefix(yearDir.Name(), "managed_user") {
					continue
				}

				// 尝试将目录名称解析为年份（4位数字）
				yearInt, err := strconv.Atoi(yearDir.Name())
				if err != nil || yearInt < 1000 || yearInt > 9999 {
					continue // 跳过非年份目录
				}

				yearPath := fmt.Sprintf("%s/%s", configUpload, yearDir.Name())

				// 按月遍历
				months, err := os.ReadDir(yearPath)
				if err != nil {
					continue
				}

				for _, monthDir := range months {
					if !monthDir.IsDir() {
						continue
					}

					monthPath := fmt.Sprintf("%s/%s/%s", configUpload, yearDir.Name(), monthDir.Name())

					// 按日遍历
					days, err := os.ReadDir(monthPath)
					if err != nil {
						continue
					}

					for _, dayDir := range days {
						if !dayDir.IsDir() {
							continue
						}

						dayPath := fmt.Sprintf("%s/%s/%s/%s", configUpload, yearDir.Name(), monthDir.Name(), dayDir.Name())

						// 读取日期目录中的文件
						files, err := os.ReadDir(dayPath)
						if err != nil {
							continue
						}

						for _, file := range files {
							if file.IsDir() || file.Name() == ".gitkeep" {
								continue
							}

							fileExt := path.Ext(file.Name())
							filePath := fmt.Sprintf("/%s/%s/%s/%s/%s", configUpload, yearDir.Name(), monthDir.Name(), dayDir.Name(), file.Name())

							// 创建文件记录
							fileRecord := models.File{
								UserId:   userInfo.ID,
								FileName: file.Name(),
								Src:      filePath,
								Ext:      fileExt,
							}

							if err := tx.Create(&fileRecord).Error; err != nil {
								return err
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	apiReturn.Success(c)
}
