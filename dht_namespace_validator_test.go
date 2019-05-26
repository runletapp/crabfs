package crabfs

import (
	"context"
	"fmt"
	"testing"
	"time"

	blocks "github.com/ipfs/go-block-format"
	"github.com/runletapp/crabfs/interfaces"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	libp2pRecord "github.com/libp2p/go-libp2p-record"
	"github.com/stretchr/testify/assert"

	pb "github.com/runletapp/crabfs/protos"
)

func setUpDHTValidatorTest(t *testing.T, pkResolver SwarmPublicKeyResolver) (libp2pRecord.Validator, *gomock.Controller) {
	ctrl := setUpBasicTest(t)
	validator := DHTNamespaceValidatorNew(context.Background(), pkResolver)

	return validator, ctrl
}

func setDownDHTValidatorTest(ctrl *gomock.Controller) {
	setDownBasicTest(ctrl)
}

func TestDHTValidatorInvalidKey(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	assert.NotNil(validator.Validate("", nil))
}

func TestDHTValidatorInvalidRecord(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	assert.NotNil(validator.Validate("/crabfs/v1/pk/hash", nil))
	assert.NotNil(validator.Validate("/crabfs/v1/pk/hash", []byte{}))
}

func TestDHTValidatorInvalidRecordTimestamp(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = ""

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	assert.NotNil(validator.Validate("/crabfs/v1/pk/hash", data))
}

func TestDHTValidatorInvalidRecordSignatureNil(t *testing.T) {
	assert := assert.New(t)
	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	record.Signature = nil

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	assert.NotNil(validator.Validate("/crabfs/v1/pk/hash", data))
}

func TestDHTValidatorInvalidRecordSignatureVerify(t *testing.T) {
	assert := assert.New(t)

	_, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	record.Signature = []byte("invalid signature")

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordSignatureVerify2(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	record.Data = []byte("data")

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	// We change the data to invalidate the signature
	record.Data = []byte("data 2")

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValue(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	record.Data = []byte("data")

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValueBlockMapNil(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = nil

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValueBlockMapZero(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = interfaces.BlockMap{}

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValueBlockMapInvalidCid(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = interfaces.BlockMap{}
	recordValue.Blocks[0] = &pb.BlockMetadata{
		Cid: nil,
	}

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValueBlockMapInvalidTotalSize(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = interfaces.BlockMap{}

	block := blocks.NewBlock([]byte("block1"))
	recordValue.Blocks[0] = &pb.BlockMetadata{
		Cid:  block.Cid().Bytes(),
		Size: int64(len(block.RawData())),
	}

	block = blocks.NewBlock([]byte("a"))
	recordValue.Blocks[1] = &pb.BlockMetadata{
		Cid: block.Cid().Bytes(),
		// Invalidate the size
		Size: int64(5),
	}

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorInvalidRecordValueBlockMapInvalidMtime(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = interfaces.BlockMap{}

	block := blocks.NewBlock([]byte("block1"))
	recordValue.Blocks[0] = &pb.BlockMetadata{
		Cid:  block.Cid().Bytes(),
		Size: int64(len(block.RawData())),
	}

	recordValue.Size = int64(len(block.RawData()))

	recordValue.Mtime = ""

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.NotNil(validator.Validate(key, data))
}

func TestDHTValidatorValid(t *testing.T) {
	assert := assert.New(t)

	privKey, pk, pkHash := GenerateKeyPairWithHash(t)

	pkResolver := func(ctx context.Context, hash string) (interfaces.PubKey, error) {
		return pk, nil
	}

	validator, ctrl := setUpDHTValidatorTest(t, pkResolver)
	defer setDownDHTValidatorTest(ctrl)

	var record pb.DHTNameRecord
	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	var recordValue pb.DHTNameRecordValue
	recordValue.Blocks = interfaces.BlockMap{}

	block := blocks.NewBlock([]byte("block1"))
	recordValue.Blocks[0] = &pb.BlockMetadata{
		Cid:  block.Cid().Bytes(),
		Size: int64(len(block.RawData())),
	}

	recordValue.Size = int64(len(block.RawData()))

	recordValue.Mtime = time.Now().UTC().Format(time.RFC3339Nano)

	var err error
	record.Data, err = proto.Marshal(&recordValue)
	assert.Nil(err)

	signature, err := privKey.Sign(record.Data)
	assert.Nil(err)
	record.Signature = signature

	data, err := proto.Marshal(&record)
	assert.Nil(err)

	key := fmt.Sprintf("/crabfs/v1/%s/hash", pkHash.String())
	assert.Nil(validator.Validate(key, data))
}

func TestDHTValidatorSelectNoValues(t *testing.T) {
	assert := assert.New(t)

	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	_, err := validator.Select("/crabfs/v1/%s/hash", [][]byte{})

	assert.NotNil(err)
}

func TestDHTValidatorSelectInvalidValues(t *testing.T) {
	assert := assert.New(t)

	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	_, err := validator.Select("/crabfs/v1/%s/hash", [][]byte{
		[]byte("abc"),
		[]byte("123"),
	})

	assert.NotNil(err)
}

func TestDHTValidatorSelectNewest(t *testing.T) {
	assert := assert.New(t)

	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	record1 := pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	record2 := pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339Nano),
	}

	data1, err := proto.Marshal(&record1)
	assert.Nil(err)

	data2, err := proto.Marshal(&record2)
	assert.Nil(err)

	n, err := validator.Select("/crabfs/v1/%s/hash", [][]byte{
		data1,
		data2,
	})

	assert.Nil(err)
	assert.Equal(1, n)
}

func TestDHTValidatorSelectNewest2(t *testing.T) {
	assert := assert.New(t)

	validator, ctrl := setUpDHTValidatorTest(t, nil)
	defer setDownDHTValidatorTest(ctrl)

	record1 := pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	record2 := pb.DHTNameRecord{
		Timestamp: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339Nano),
	}

	data1, err := proto.Marshal(&record1)
	assert.Nil(err)

	data2, err := proto.Marshal(&record2)
	assert.Nil(err)

	n, err := validator.Select("/crabfs/v1/%s/hash", [][]byte{
		data2,
		data1,
	})

	assert.Nil(err)
	assert.Equal(0, n)
}
