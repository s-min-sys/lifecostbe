package storage

import "github.com/s-min-sys/lifecostbe/internal/model"

func (impl *storageImpl) initData() (err error) {
	earnID, _, err := impl.NewPerson("[全局]挣钱了")
	if err != nil {
		return
	}

	err = impl.SetPersonMerchant(earnID, model.CostDirOut)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("银行利息", earnID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("工资/薪水", earnID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("理财", earnID)
	if err != nil {
		return
	}

	//
	//
	//

	spendID, _, err := impl.NewPerson("[全局]消费了")
	if err != nil {
		return
	}

	err = impl.SetPersonMerchant(spendID, model.CostDirIn)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("超市", spendID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("菜市场", spendID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("公交", spendID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("订餐", spendID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("外出就餐", spendID)
	if err != nil {
		return
	}

	_, err = impl.NewWallet("游玩", spendID)
	if err != nil {
		return
	}

	//
	//
	//

	_, err = impl.NewLabel("大额支出")
	if err != nil {
		return
	}

	return err
}
