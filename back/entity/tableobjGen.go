package tableobj

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func tableobjGen() {
	// 创建一个生成器实例
	g := gen.NewGenerator(gen.Config{
		ModelPkgPath:  "tableobj",   // 设置为空字符串，避免生成 model 文件夹
		OutPath:       "./tableobj", // 输出路径
		OutFile:       ".go",
		Mode:          gen.WithDefaultQuery | gen.WithoutContext, // 设置生成模式
		FieldNullable: true,                                      // 开启字段可空支持
	})

	// 设置数据库连接
	dsn := "root:zhangyw12@tcp(47.236.89.82:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
	t, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	g.UseDB(t) // 传递 DSN 给生成器

	// 执行生成任务
	g.GenerateAllTable() // 生成所有表的模型

	// 执行写入文件操作
	g.Execute()

	// 指定要处理的目录
	dir := g.OutPath

	// 获取目录下的文件列表
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	// 遍历文件列表
	for _, file := range files {
		// 检查是否是文件（而不是目录）
		if !file.IsDir() {
			oldName := file.Name()
			newName := strings.Replace(oldName, ".gen", "", -1)

			// 如果文件名发生了变化，进行重命名
			if oldName != newName {
				oldPath := dir + "/" + oldName
				newPath := dir + "/" + newName

				err := os.Rename(oldPath, newPath)
				if err != nil {
					log.Printf("Failed to rename file %s to %s: %v", oldName, newName, err)
				} else {
					fmt.Printf("Renamed file %s to %s\n", oldName, newName)
				}
			}
		}
	}
}
