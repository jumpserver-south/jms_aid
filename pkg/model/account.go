package model

type AssetAccount struct {
	AssetId      string `json:"asset_id"`
	SystemuserId string `json:"systemuser_id"`
	Name         string `json:"name" gorm:"name"`
	Username     string `json:"username" gorm:"username"`
	Password     string `json:"password" gorm:"password"`
	PrivateKey   string `json:"private_key" gorm:"private_key"`
	PublicKey    string `json:"public_key" gorm:"public_key"`
}
