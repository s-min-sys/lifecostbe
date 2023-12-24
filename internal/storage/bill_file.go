package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/godruoyi/go-snowflake"
	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/pathutils"
	"golang.org/x/exp/slices"
)

type BillFile interface {
	AddBill(bill model.GroupBill) error
	GetBill(billID string) (bill model.GroupBill, err error)
	DeleteRecord(billID string) (err error)
	GetBills(startDate, finishDate string) ([]model.GroupBill, error)
	ListBills(id string, count int, dirNew bool) (bills []model.GroupBill, hasMore bool, err error)
}

func NewBillFile(dir string, base string, logger l.Wrapper) BillFile {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return &billFileImpl{
		dir:    dir,
		base:   base,
		logger: logger.WithFields(l.StringField(l.ClsKey, "billFileImpl")),
		files:  make(map[string]*streamFile),
	}
}

type streamFile struct {
	lock           sync.Mutex
	file           *os.File
	key            string
	filePath       string
	latestRecordAt time.Time
	lastAccessAt   time.Time
}

type billFileImpl struct {
	lock   sync.Mutex
	dir    string
	base   string
	logger l.Wrapper

	files map[string]*streamFile
}

func (impl *billFileImpl) getFileKey(at time.Time) string {
	return at.Format("20060102")
}

func (impl *billFileImpl) getFileName(date8 string) string {
	return filepath.Join(impl.base + "-" + date8)
}

func (impl *billFileImpl) getFilePath(key string) string {
	return filepath.Join(impl.dir, impl.getFileName(key))
}

func (impl *billFileImpl) getFileAt(at time.Time) (file *streamFile, err error) {
	return impl.getFileByKey(impl.getFileKey(at))
}

func (impl *billFileImpl) getFileByKey(key string) (file *streamFile, err error) {
	impl.lock.Lock()

	defer impl.lock.Unlock()

	file, ok := impl.files[key]
	if ok {
		file.lastAccessAt = time.Now()

		return
	}

	filePath := impl.getFilePath(key)

	_ = pathutils.MustDirOfFileExists(filePath)

	bills, _ := impl.readFileBills(filePath)

	latestRecordAt := time.Now()

	_ = os.RemoveAll(filePath)

	rawFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return
	}

	if len(bills) > 0 {
		slices.SortFunc(bills, func(a, b model.GroupBill) int {
			if a.At == b.At {
				return 0
			}

			if a.At < b.At {
				return -1
			}

			return 1
		})

		latestRecordAt, err = impl.writeAllBillsOnFile(rawFile, bills)
	}

	file = &streamFile{
		file:           rawFile,
		key:            key,
		filePath:       filePath,
		latestRecordAt: latestRecordAt,
		lastAccessAt:   time.Now(),
	}

	impl.files[key] = file

	return
}

func (impl *billFileImpl) AddBill(bill model.GroupBill) error {
	if !bill.Valid() {
		return commerr.ErrInvalidArgument
	}

	at := time.Unix(bill.At, 0)

	bill.ID = fmt.Sprintf("%s%d", at.Format("20060102"), snowflake.ID())

	sf, err := impl.getFileAt(at)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.AnyField("at", at)).Error("get File failed")

		return commerr.ErrInternal
	}

	sf.lock.Lock()
	defer sf.lock.Unlock()

	if sf.latestRecordAt.Before(at) {
		var d []byte

		d, err = json.Marshal(bill)
		if err != nil {
			impl.logger.WithFields(l.ErrorField(err)).Error("marshal bill failed")

			return err
		}

		line := string(d) + "\n"

		_, err = sf.file.Write([]byte(line))
		if err != nil {
			impl.logger.WithFields(l.ErrorField(err)).Error("write file failed")

			return err
		}

		sf.latestRecordAt = at

		return nil
	}

	_ = sf.file.Close()

	bills, err := impl.readFileBills(sf.filePath)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.StringField("filePath", sf.filePath)).
			Error("read bills failed")

		return err
	}

	bills = append(bills, bill)

	_ = os.RemoveAll(sf.filePath)

	slices.SortFunc(bills, func(a, b model.GroupBill) int {
		if a.At == b.At {
			return 0
		}

		if a.At < b.At {
			return -1
		}

		return 1
	})

	rawFile, err := os.OpenFile(sf.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		delete(impl.files, sf.key)

		impl.logger.WithFields(l.ErrorField(err), l.StringField("file", sf.filePath)).
			Error("reopen file failed")

		return err
	}

	sf.file = rawFile

	sf.latestRecordAt, err = impl.writeAllBillsOnFile(sf.file, bills)

	return err
}

func (impl *billFileImpl) GetBill(billID string) (bill model.GroupBill, err error) {
	if len(billID) <= 8 {
		err = commerr.ErrInvalidArgument

		return
	}

	key := billID[:8]

	sf, err := impl.getFileByKey(key)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.AnyField("key", key)).Error("get File failed")

		err = commerr.ErrInternal

		return
	}

	bills, err := impl.readFileBills(sf.filePath)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.StringField("filePath", sf.filePath)).
			Error("read bills failed")

		return
	}

	for _, groupBill := range bills {
		if groupBill.ID == billID {
			bill = groupBill

			return
		}
	}

	err = commerr.ErrNotFound

	return
}

func (impl *billFileImpl) DeleteRecord(billID string) (err error) {
	if len(billID) <= 8 {
		err = commerr.ErrInvalidArgument

		return
	}

	key := billID[:8]

	sf, err := impl.getFileByKey(key)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.AnyField("key", key)).Error("get File failed")

		return commerr.ErrInternal
	}

	sf.lock.Lock()
	defer sf.lock.Unlock()

	_ = sf.file.Close()

	bills, err := impl.readFileBills(sf.filePath)
	if err != nil {
		impl.logger.WithFields(l.ErrorField(err), l.StringField("filePath", sf.filePath)).
			Error("read bills failed")

		return err
	}

	_ = os.RemoveAll(sf.filePath)

	for idx := 0; idx < len(bills); idx++ {
		if bills[idx].ID == billID {
			bills = slices.Delete(bills, idx, idx+1)

			break
		}
	}

	slices.SortFunc(bills, func(a, b model.GroupBill) int {
		if a.At == b.At {
			return 0
		}

		if a.At < b.At {
			return -1
		}

		return 1
	})

	rawFile, err := os.OpenFile(sf.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		delete(impl.files, sf.key)

		impl.logger.WithFields(l.ErrorField(err), l.StringField("file", sf.filePath)).
			Error("reopen file failed")

		return err
	}

	sf.file = rawFile

	sf.latestRecordAt, err = impl.writeAllBillsOnFile(sf.file, bills)

	return
}

func (impl *billFileImpl) writeAllBillsOnFile(file *os.File, bills []model.GroupBill) (latestRecordAt time.Time,
	err error) {
	var d []byte

	for _, b := range bills {
		d, err = json.Marshal(b)
		if err != nil {
			return
		}

		line := string(d) + "\n"

		_, err = file.Write([]byte(line))
		if err != nil {
			return
		}

		latestRecordAt = time.Unix(b.At, 0)
	}

	return
}

func (impl *billFileImpl) readFileBills(path string) (bills []model.GroupBill, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		var line []byte

		line, err = reader.ReadBytes('\n')
		if err == io.EOF {
			err = nil

			break
		}

		if err != nil {
			return
		}

		var bill model.GroupBill

		err = json.Unmarshal(line, &bill)
		if err != nil {
			return
		}

		bills = append(bills, bill)
	}

	return
}

func (impl *billFileImpl) GetBills(startDate, finishDate string) (bills []model.GroupBill, err error) {
	if len(startDate) != 0 && len(startDate) != 8 {
		err = commerr.ErrInvalidArgument

		return
	}

	if len(finishDate) != 0 && len(finishDate) != 8 {
		err = commerr.ErrInvalidArgument

		return
	}

	var startDateFileName, finishDateFileName string

	if len(startDate) != 0 {
		startDateFileName = impl.base + "-" + startDate
	}

	if len(finishDate) != 0 {
		finishDateFileName = impl.base + "-" + finishDate
	}

	files, err := os.ReadDir(impl.dir)
	if err != nil {
		return
	}

	billFiles := make([]string, 0, 10)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasPrefix(file.Name(), impl.base) {
			continue
		}

		if startDateFileName != "" {
			if file.Name() < startDateFileName {
				continue
			}
		}

		if finishDateFileName != "" {
			if file.Name() > finishDateFileName {
				continue
			}
		}

		billFiles = append(billFiles, file.Name())
	}

	slices.Sort(billFiles)

	for _, file := range billFiles {
		var dayBills []model.GroupBill

		dayBills, err = impl.readFileBills(filepath.Join(impl.dir, file))
		if err != nil {
			return
		}

		bills = append(bills, dayBills...)
	}

	return
}

func (impl *billFileImpl) ListBills(id string, count int, dirNew bool) (bills []model.GroupBill, hasMore bool, err error) {
	files, err := os.ReadDir(impl.dir)
	if err != nil {
		return
	}

	if count > 0 {
		count++
	}

	var inFileName string

	if id != "" && len(id) > 8 {
		inFileName = impl.getFileName(id[:8])
	}

	billFiles := make([]string, 0, 10)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasPrefix(file.Name(), impl.base) {
			continue
		}

		if inFileName != "" {
			if dirNew {
				if file.Name() < inFileName {
					continue
				}
			} else {
				if file.Name() > inFileName {
					continue
				}
			}
		}

		billFiles = append(billFiles, file.Name())
	}

	slices.SortFunc(billFiles, func(a, b string) int {
		r := strings.Compare(a, b)
		if !dirNew {
			r = -r
		}

		return r
	})

	/*
		d: 3 2 1
		u: 1 2 3
	*/

	for x, file := range billFiles {
		if count > 0 && len(bills) >= count {
			break
		}

		var dayBills []model.GroupBill

		dayBills, err = impl.readFileBills(filepath.Join(impl.dir, file))
		if err != nil {
			return
		}

		if x == 0 {
			for y, bill := range dayBills {
				if bill.ID == id {
					if dirNew {
						dayBills = dayBills[y+1:]
					} else {
						dayBills = dayBills[:y]
					}

					break
				}
			}
		}

		if dirNew {
			for y := 0; y < len(dayBills); y++ {
				bills = append(bills, dayBills[y])

				if count > 0 && len(bills) >= count {
					break
				}
			}
		} else {
			for y := len(dayBills) - 1; y >= 0; y-- {
				bills = append(bills, dayBills[y])

				if count > 0 && len(bills) >= count {
					break
				}
			}
		}
	}

	if count > 0 {
		if len(bills) >= count {
			bills = bills[:len(bills)-1]
			hasMore = true
		}
	}

	return
}
