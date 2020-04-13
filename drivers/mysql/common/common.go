package common

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/siddontang/go-mysql/mysql"
	"strconv"
	"strings"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
)

func init() {
	consul.Register()
}

type StoreManager struct {
	consulStore store.Store
}
func NewStoreManager(consulAddr []string) (*StoreManager, error) {
	consulStore, err := libkv.NewStore(store.CONSUL, consulAddr, nil)
	if err != nil {
		return nil, err
	}
	return &StoreManager{consulStore:consulStore}, nil
}
func (sm *StoreManager) DestroyJob(jobName string) error {
	key := fmt.Sprintf("dtle/%v", jobName)
	err := sm.consulStore.DeleteTree(key)
	if err == store.ErrKeyNotFound {
		return nil
	} else {
		return err
	}
}
func (sm *StoreManager) SaveGtidForJob(jobName string, gtid string) error {
	key := fmt.Sprintf("dtle/%v/Gtid", jobName)
	err := sm.consulStore.Put(key, []byte(gtid), nil)
	return err
}
const binlogFilePosSeparator = "//dtle//"
func (sm *StoreManager) SaveBinlogFilePosForJob(jobName string, file string, pos int) error {
	key := fmt.Sprintf("dtle/%v/BinlogFilePos", jobName)
	s := fmt.Sprintf("%v%v%v", file, binlogFilePosSeparator, pos)
	err := sm.consulStore.Put(key, []byte(s), nil)
	return err
}
func (sm *StoreManager) GetBinlogFilePosForJob(jobName string) (*mysql.Position, error) {
	key := fmt.Sprintf("dtle/%v/BinlogFilePos", jobName)
	p, err := sm.consulStore.Get(key)
	if err == store.ErrKeyNotFound {
		return &mysql.Position{
			Name: "",
			Pos:  0,
		}, nil
	} else if err != nil {
		return nil, err
	}
	s := string(p.Value)
	ss := strings.Split(s, binlogFilePosSeparator)
	if len(ss) != 2 {
		return nil, fmt.Errorf("Unexpected BinlogFilePos format. value %v", s)
	}
	pos, err := strconv.Atoi(ss[1])
	if err != nil {
		return nil, errors.Wrap(err, "Atoi")
	}
	return &mysql.Position{
		Name: ss[0],
		Pos:  uint32(pos),
	}, nil
}
func (sm *StoreManager) GetGtidForJob(jobName string) (string, error) {
	key := fmt.Sprintf("dtle/%v/Gtid", jobName)
	p, err := sm.consulStore.Get(key)
	if err == store.ErrKeyNotFound {
		return "", nil
	} else if err != nil {
		return "", err
	}
	// Get a non-existing KV
	return string(p.Value), nil
}

func (sm *StoreManager) PutNats(jobName string, natsAddr string) error {
	natsKey := fmt.Sprintf("dtle/%v/NatsAddr", jobName)
	return sm.consulStore.Put(natsKey, []byte(natsAddr), nil)
}

func (sm *StoreManager) WatchNats(jobName string, stopCh chan struct{}) (string, error) {
	natsKey := fmt.Sprintf("dtle/%v/NatsAddr", jobName)
	natsCh, err := sm.consulStore.Watch(natsKey, stopCh)
	if err != nil {
		return "", err
	}
	kv := <-natsCh
	return string(kv.Value), nil
}