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
	GetBills(startDate, finishDate string) ([]model.GroupBill, error)
	GetBillsByID(id string, count int, dirNew bool) (bills []model.GroupBill, hasMore bool, err error)
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
	lock         sync.Mutex
	file         *os.File
	lastAccessAt time.Time
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

func (impl *billFileImpl) getFilePath(at time.Time) string {
	return filepath.Join(impl.dir, impl.getFileName(at.Format("20060102")))
}

func (impl *billFileImpl) getFile(at time.Time) *streamFile {
	impl.lock.Lock()

	defer impl.lock.Unlock()

	key := impl.getFileKey(at)

	file, ok := impl.files[key]
	if ok {
		file.lastAccessAt = time.Now()

		return file
	}

	filePath := impl.getFilePath(at)

	_ = pathutils.MustDirOfFileExists(filePath)

	rawFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil
	}

	impl.files[key] = &streamFile{
		file:         rawFile,
		lastAccessAt: time.Now(),
	}

	return impl.files[key]
}

func (impl *billFileImpl) AddBill(bill model.GroupBill) error {
	if !bill.Valid() {
		return commerr.ErrInvalidArgument
	}

	at := time.Unix(bill.At, 0)

	bill.ID = fmt.Sprintf("%s%d", at.Format("20060102"), snowflake.ID())

	sf := impl.getFile(at)
	if sf == nil {
		return commerr.ErrInternal
	}

	d, err := json.Marshal(bill)
	if err != nil {
		return err
	}

	line := string(d) + "\n"

	sf.lock.Lock()
	defer sf.lock.Unlock()

	_, err = sf.file.Write([]byte(line))

	return err
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

func (impl *billFileImpl) GetBillsByID(id string, count int, dirNew bool) (bills []model.GroupBill, hasMore bool, err error) {
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
