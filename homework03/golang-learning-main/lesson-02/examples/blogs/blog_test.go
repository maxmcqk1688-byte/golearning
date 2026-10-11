package blogs

import (
	"fmt"
	"lesson02examples/testutil"
	"log"
	"testing"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username  string `gorm:"type:varchar(50);not null;uniqueIndex"`
	Email     string `gorm:"type:varchar(100);not null;uniqueIndex"`
	PostCount int    `gorm:"default:0;not null"` // 用户文章数量统计字段
	// 一对多关系：一个 User 拥有多篇 Post
	Posts []Post `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Post 文章模型
type Post struct {
	gorm.Model
	Title         string `gorm:"type:varchar(200);not null"`
	Content       string `gorm:"type:text;not null"`
	CommentStatus string `gorm:"type:varchar(20);default:'无评论';not null"` // 评论状态字段
	// 关联 User：外键与反向引用
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"foreignKey:UserID"`

	// 一对多关系：一篇 Post 拥有多条 Comment
	Comments []Comment `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Comment 评论模型
type Comment struct {
	gorm.Model
	Content string `gorm:"type:text;not null"`

	// 关联 Post：外键与反向引用
	PostID uint `gorm:"not null;index"`
	Post   Post `gorm:"foreignKey:PostID"`

	// 评论发布者（可选）：通常评论也需要关联具体的 User
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"foreignKey:UserID"`
}

func GetUserPostsWithComments(db *gorm.DB, userID uint) (*User, error) {
	var user User

	// 方式一：从 User 出发，嵌套预加载 Posts 及其 Comments
	err := db.Preload("Posts.Comments").
		First(&user, userID).Error

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 补充方式二：直接查询文章列表（如果不需要携带用户信息本体，这种方式更为轻量）
func GetPostsByUserIDWithComments(db *gorm.DB, userID uint) ([]Post, error) {
	var posts []Post

	err := db.Preload("Comments").
		Where("user_id = ?", userID).
		Find(&posts).Error

	return posts, err
}

// PostWithCommentCount 带有评论统计数量的结果结构体
type PostWithCommentCount struct {
	Post
	CommentCount int64 `json:"comment_count"`
}

// GetMostCommentedPost 查询评论数量最多的文章信息（包含评论总数）
func GetMostCommentedPost(db *gorm.DB) (*PostWithCommentCount, error) {
	var result PostWithCommentCount

	// 使用 left join 避免忽略没有任何评论的文章，聚合后按数量倒序取第 1 条
	err := db.Table("posts").
		Select("posts.*, COUNT(comments.id) as comment_count").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.deleted_at IS NULL").
		Where("posts.deleted_at IS NULL"). // 兼容 gorm.Model 的软删除
		Group("posts.id").
		Order("comment_count DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// 如果只需要标准的 Post 模型对象
func GetTopPostOnly(db *gorm.DB) (*Post, error) {
	var post Post

	err := db.Model(&Post{}).
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.deleted_at IS NULL").
		Group("posts.id").
		Order("COUNT(comments.id) DESC").
		Limit(1).
		First(&post).Error

	return &post, err
}

// ======================== 钩子函数 (Hooks) ========================

// 1. Post 的 AfterCreate 钩子：文章创建后，自动递增用户的文章统计数量
func (p *Post) AfterCreate(tx *gorm.DB) (err error) {
	// 使用 gorm.Expr 进行原子自增，避免并发更新冲突
	err = tx.Model(&User{}).
		Where("id = ?", p.UserID).
		UpdateColumn("post_count", gorm.Expr("post_count + ?", 1)).
		Error
	if err != nil {
		return fmt.Errorf("更新用户文章数量失败: %w", err)
	}
	return nil
}

// 2. Comment 的 AfterDelete 钩子：评论删除后，检查文章剩余评论数
func (c *Comment) AfterDelete(tx *gorm.DB) (err error) {
	var count int64

	// 统计当前文章下未被软删除的评论数量
	err = tx.Model(&Comment{}).
		Where("post_id = ?", c.PostID).
		Count(&count).
		Error
	if err != nil {
		return fmt.Errorf("统计剩余评论数量失败: %w", err)
	}

	// 如果没有剩余评论，将状态修改为 "无评论"
	if count == 0 {
		err = tx.Model(&Post{}).
			Where("id = ?", c.PostID).
			Update("comment_status", "无评论").
			Error
		if err != nil {
			return fmt.Errorf("更新文章评论状态失败: %w", err)
		}
	}

	return nil
}

func TestBlog(t *testing.T) {

	db := testutil.NewTestDB(t, "blog.db")

	err := db.AutoMigrate(&User{}, &Post{}, &Comment{})
	if err != nil {
		log.Fatalf("数据表迁移失败: %v", err)
	}

	log.Println("数据表创建成功！已生成 users, posts, comments 表。")

	// 1. 创建测试用户
	user := User{Username: "Alice", Email: "alice@example.com"}
	db.Create(&user)
	fmt.Printf("初始用户文章数: %d\n", user.PostCount)

	// 2. 测试 Post 钩子：创建文章
	post := Post{
		Title:         "GORM Hooks 实践",
		Content:       "深入浅出 GORM 生命周期钩子",
		UserID:        user.ID,
		CommentStatus: "有评论", // 初始假定有评论以便观察删除后的状态更新
	}
	db.Create(&post)

	// 查询更新后的 User
	var updatedUser User
	db.First(&updatedUser, user.ID)
	fmt.Printf("创建一篇文章后，用户文章数: %d\n", updatedUser.PostCount)

	// 3. 为文章添加一条评论
	comment := Comment{
		Content: "写得不错！",
		PostID:  post.ID,
		UserID:  user.ID,
	}
	db.Create(&comment)

	comment1 := Comment{
		Content: "写得很好！",
		PostID:  post.ID,
		UserID:  user.ID,
	}
	db.Create(&comment1)

	// 4. 测试 Comment 钩子：删除该评论
	db.Delete(&comment)

	// 查询删除评论后的 Post 状态
	var updatedPost Post
	db.First(&updatedPost, post.ID)
	fmt.Printf("删除全部评论后，文章评论状态: %s\n", updatedPost.CommentStatus)

	// ------------------ 测试需求 1 ------------------
	
	user1, err := GetUserPostsWithComments(db, user.ID)
	if err != nil {
		log.Printf("查询用户 %d 的文章与评论失败: %v\n", user.ID, err)
	} else {
		fmt.Printf("用户: %s (ID: %d)\n", user1.Username, user1.ID)
		for _, post := range user.Posts {
			fmt.Printf("  ├── 文章: %s (评论数: %d)\n", post.Title, len(post.Comments))
			for _, comment2 := range post.Comments {
				fmt.Printf("  │     └── 评论: %s\n", comment2.Content)
			}
		}
	}

	// ------------------ 测试需求 2 ------------------
	topPost, err := GetMostCommentedPost(db)
	if err != nil {
		log.Printf("查询评论最多的文章失败: %v\n", err)
	} else {
		fmt.Printf("\n评论最多的文章: %s, 评论总数: %d 条\n", topPost.Title, topPost.CommentCount)
	}
}
