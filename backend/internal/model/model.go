// Package model 定义本地电气台账的核心数据模型。
package model

// Room 配电室。
type Room struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Remark       string  `json:"remark"`
	CabinetCount int     `json:"cabinet_count"`
	ImageIDs     []int64 `json:"image_ids"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// Cabinet 配电柜，属于某个配电室。
type Cabinet struct {
	ID        int64   `json:"id"`
	RoomID    int64   `json:"room_id"`
	RoomName  string  `json:"room_name"`
	Name      string  `json:"name"`
	Remark    string  `json:"remark"`
	ImageIDs  []int64 `json:"image_ids"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// Equipment 台账记录（总表的一行）。
type Equipment struct {
	ID           int64   `json:"id"`
	RoomID       int64   `json:"room_id"`
	RoomName     string  `json:"room_name"`
	CabinetID    *int64  `json:"cabinet_id"`
	CabinetName  string  `json:"cabinet_name"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	Manufacturer string  `json:"manufacturer"`
	Quantity     int     `json:"quantity"`
	Remark       string  `json:"remark"`
	ImageIDs     []int64 `json:"image_ids"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// Annex 附件表，一个附件对应磁盘上 ./data/annex/<uuid>.<ext> 的一个文件。
// RefCount 为引用次数（冗余列），由 annex_ref 可随时重算。
type Annex struct {
	ID           int64  `json:"id"`
	UUID         string `json:"uuid"`
	OriginalName string `json:"original_name"`
	Ext          string `json:"ext"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	RefCount     int    `json:"ref_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	URL          string `json:"url,omitempty"`
}

// AnnexRefTarget 附件引用的目标类型。
type AnnexRefTarget string

const (
	TargetRoom      AnnexRefTarget = "room"
	TargetCabinet   AnnexRefTarget = "cabinet"
	TargetEquipment AnnexRefTarget = "equipment"
)

// EquipmentListResult 分页列表结果。
type EquipmentListResult struct {
	Items    []Equipment `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// RoomInput 创建/更新配电室的请求体。
type RoomInput struct {
	Name     string  `json:"name"`
	Remark   string  `json:"remark"`
	ImageIDs []int64 `json:"image_ids"`
}

// CabinetInput 创建/更新配电柜的请求体。
type CabinetInput struct {
	RoomID   int64   `json:"room_id"`
	Name     string  `json:"name"`
	Remark   string  `json:"remark"`
	ImageIDs []int64 `json:"image_ids"`
}

// EquipmentInput 创建/更新台账记录的请求体。
type EquipmentInput struct {
	RoomID       int64   `json:"room_id"`
	CabinetID    *int64  `json:"cabinet_id"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	Manufacturer string  `json:"manufacturer"`
	Quantity     int     `json:"quantity"`
	Remark       string  `json:"remark"`
	ImageIDs     []int64 `json:"image_ids"`
}
