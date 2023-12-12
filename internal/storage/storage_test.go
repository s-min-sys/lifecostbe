package storage

import (
	"os"
	"testing"
	"time"

	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

var utWorkDir = "../../uts/"

func TestMain(m *testing.M) {
	_ = os.MkdirAll(utWorkDir, os.ModePerm)
	_ = os.Chdir(utWorkDir)

	code := m.Run()

	_ = os.Chdir("..")

	_ = os.RemoveAll("uts")

	os.Exit(code)
}

func utEqualSliceIgnoreOrder(t *testing.T, a, b []uint64) {
	assert.EqualValues(t, len(a), len(b))
	slices.Sort(a)
	slices.Sort(b)
	assert.EqualValues(t, a, b)
}

func TestStorage(t *testing.T) {

	_ = os.RemoveAll("organization")
	stg := NewStorage(".", false)

	//
	// create persons
	//

	zjzPersonID, zjzDefaultWalletID, err := stg.NewPerson("zjz")
	assert.Nil(t, err)
	assert.True(t, zjzPersonID > 0)
	assert.True(t, zjzDefaultWalletID > 0)

	zymPersonID, zymDefaultWalletID, err := stg.NewPerson("zym")
	assert.Nil(t, err)
	assert.True(t, zymPersonID > 0)
	assert.True(t, zymDefaultWalletID > 0)

	zzxPersonID, zzxDefaultWalletID, err := stg.NewPerson("zzx")
	assert.Nil(t, err)
	assert.True(t, zzxPersonID > 0)
	assert.True(t, zzxDefaultWalletID > 0)

	jinBaoPersonID, jinBaoDefaultWalletID, err := stg.NewPerson("jinBao")
	assert.Nil(t, err)
	assert.True(t, jinBaoPersonID > 0)
	assert.True(t, jinBaoDefaultWalletID > 0)

	//
	// create home group
	//

	homeGroupID, err := stg.NewGroup("home", zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, homeGroupID > 0)

	personIDs, adminPersonIDs, err := stg.GetGroupPersonIDs(homeGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID}, adminPersonIDs)

	groupIDs, err := stg.GetPersonGroupsIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID}, groupIDs)

	groupIDs, err = stg.GetPersonGroupsIDs(zymPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, nil, groupIDs)

	err = stg.JoinGroup(homeGroupID, zymPersonID)
	assert.Nil(t, err)

	err = stg.JoinGroup(homeGroupID, zzxPersonID)
	assert.Nil(t, err)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(homeGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID}, adminPersonIDs)

	groupIDs, err = stg.GetPersonGroupsIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID}, groupIDs)

	groupIDs, err = stg.GetPersonGroupsIDs(zymPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID}, groupIDs)

	adminFlag, err := stg.IsGroupAdmin(homeGroupID, zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zymPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zzxPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	err = stg.SetGroupAdmin(homeGroupID, zymPersonID, true)
	assert.Nil(t, err)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(homeGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zymPersonID}, adminPersonIDs)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zymPersonID)
	assert.Nil(t, err)
	assert.True(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zzxPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	err = stg.SetGroupAdmin(homeGroupID, zjzPersonID, false)
	assert.Nil(t, err)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zjzPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zymPersonID)
	assert.Nil(t, err)
	assert.True(t, adminFlag)

	adminFlag, err = stg.IsGroupAdmin(homeGroupID, zzxPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(homeGroupID)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zymPersonID}, adminPersonIDs)

	//
	// create jiaZu group
	//

	jiaZuGroupID, err := stg.NewGroup("jiaZu", zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, jiaZuGroupID > 0)

	err = stg.JoinGroup(jiaZuGroupID, zymPersonID)
	assert.Nil(t, err)

	err = stg.JoinGroup(jiaZuGroupID, zzxPersonID)
	assert.Nil(t, err)

	err = stg.SetGroupAdmin(jiaZuGroupID, jinBaoPersonID, true)
	assert.NotNil(t, err)

	err = stg.JoinGroup(jiaZuGroupID, jinBaoPersonID)
	assert.Nil(t, err)

	err = stg.SetGroupAdmin(jiaZuGroupID, jinBaoPersonID, true)
	assert.Nil(t, err)

	adminFlag, err = stg.IsGroupAdmin(jiaZuGroupID, jinBaoPersonID)
	assert.Nil(t, err)
	assert.True(t, adminFlag)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(jiaZuGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID, jinBaoPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, jinBaoPersonID}, adminPersonIDs)

	groupIDs, err = stg.GetPersonGroupsIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID, jiaZuGroupID}, groupIDs)

	err = stg.LeaveGroup(jiaZuGroupID, jinBaoPersonID)
	assert.Nil(t, err)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(jiaZuGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID}, adminPersonIDs)

	groupIDs, err = stg.GetPersonGroupsIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID, jiaZuGroupID}, groupIDs)

	adminFlag, err = stg.IsGroupAdmin(jiaZuGroupID, jinBaoPersonID)
	assert.NotNil(t, err)

	err = stg.JoinGroup(jiaZuGroupID, jinBaoPersonID)
	assert.Nil(t, err)

	adminFlag, err = stg.IsGroupAdmin(jiaZuGroupID, jinBaoPersonID)
	assert.Nil(t, err)
	assert.False(t, adminFlag)

	groupIDs, err = stg.GetPersonGroupsIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{homeGroupID, jiaZuGroupID}, groupIDs)

	personIDs, adminPersonIDs, err = stg.GetGroupPersonIDs(jiaZuGroupID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID, zzxPersonID, zymPersonID, jinBaoPersonID}, personIDs)
	utEqualSliceIgnoreOrder(t, []uint64{zjzPersonID}, adminPersonIDs)

	subWeChatWalletID, err := stg.NewWallet("wechat", zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, subWeChatWalletID > 0)

	subBankWalletID, err := stg.NewWallet("bank", zjzPersonID)
	assert.Nil(t, err)
	assert.True(t, subBankWalletID > 0)

	subWalletIDs, err := stg.GetPersonWalletIDs(zjzPersonID)
	assert.Nil(t, err)
	utEqualSliceIgnoreOrder(t, []uint64{subBankWalletID, subWeChatWalletID, zjzPersonID}, subWalletIDs)

	//
	//
	//
	bankPersonID, _, err := stg.NewPerson("Bank")
	assert.Nil(t, err)
	assert.True(t, bankPersonID > 0)

	liXiSubWalletID, err := stg.NewWallet("利息", bankPersonID)
	assert.Nil(t, err)
	assert.True(t, liXiSubWalletID > 0)

	markPersonID, _, err := stg.NewPerson("Mark")
	assert.Nil(t, err)
	assert.True(t, markPersonID > 0)

	xiaoFeiSubWalletID, err := stg.NewWallet("消费", markPersonID)
	assert.Nil(t, err)
	assert.True(t, xiaoFeiSubWalletID > 0)

	//
	//
	//

	huaFeiLabelID, err := stg.NewLabel("日常花费")

	err = stg.Record(homeGroupID, model.GroupBill{
		FromSubWalletID: subWeChatWalletID,
		ToSubWalletID:   xiaoFeiSubWalletID,
		CostDir:         model.CostDirOut,
		Amount:          10000,
		LabelIDs:        []uint64{huaFeiLabelID},
		Remark:          "在超市里，我微信花费100块",
		At:              time.Now().Unix(),
	})
	assert.Nil(t, err)

	err = stg.Record(homeGroupID, model.GroupBill{
		FromSubWalletID: liXiSubWalletID,
		ToSubWalletID:   subWeChatWalletID,
		CostDir:         model.CostDirIn,
		Amount:          2000,
		LabelIDs:        []uint64{huaFeiLabelID},
		Remark:          "银行利息20块到手了",
		At:              time.Now().Unix(),
	})
	assert.Nil(t, err)

	bills, err := stg.GetBills(homeGroupID)
	assert.Nil(t, err)

	t.Log(bills)
}
