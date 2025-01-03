package bloomFilter

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spaolacci/murmur3"
	"golang.org/x/crypto/blake2b"
	"strconv"
	"time"
)

type BloomFilter struct {
	cmd       redis.Cmdable
	bitmapKey string
}

func NewBloomFilter(cmd redis.Cmdable, bitmapKey string) *BloomFilter {
	return &BloomFilter{
		cmd:       cmd,
		bitmapKey: bitmapKey,
	}
}

func (b *BloomFilter) GetMurmur3(key string) (uint64, error) {
	// 将手机号转为uint64类型
	p, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("手机号转换失败，%w", err)
	}
	// 创建Murmur3的实例对象
	obj := murmur3.New64()
	// 将手机号写入hash对象
	binary.Write(obj, binary.BigEndian, p)
	// 计算Hash值
	hash := obj.Sum64()
	return hash, nil
}

func (b *BloomFilter) GetMD5(key string) uint64 {
	// 创建MD5的实例对象
	obj := md5.New()
	// 获取md5哈希值
	bytes := obj.Sum([]byte(key))
	// 将MD5值转为uint64
	res := binary.BigEndian.Uint64(bytes)
	return res
}

func (b *BloomFilter) GetBLAKE2(key string) (uint64, error) {
	// 创建blake2b的实例对象
	new256, err := blake2b.New256([]byte(key))
	// 计算哈希值
	bytes := new256.Sum(nil)
	// 将哈希值转为uint64
	res := binary.BigEndian.Uint64(bytes)
	return res, err
}

func (b *BloomFilter) SetMurmur3BitMap(key string) error {
	murmur3, err := b.GetMurmur3(key)
	if err != nil {
		return err
	}
	murmur3 = murmur3 % (1 << 32) // 取模 2^32
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 创建redis的bitmap
	err = b.cmd.SetBit(ctx, b.bitmapKey, int64(murmur3), 1).Err()
	if err != nil {
		return err
	}
	return nil
}

func (b *BloomFilter) SetMD5BitMap(key string) error {
	md5 := b.GetMD5(key) % (1 << 32)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 创建redis的bitmap
	err := b.cmd.SetBit(ctx, b.bitmapKey, int64(md5), 1).Err()
	if err != nil {
		return err
	}
	return nil
}

func (b *BloomFilter) SetBLAKE2BitMap(key string) error {
	blake2, err := b.GetBLAKE2(key)
	if err != nil {
		return err
	}
	blake2 = blake2 % (1 << 32)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 创建redis的bitmap
	err = b.cmd.SetBit(ctx, b.bitmapKey, int64(blake2), 1).Err()
	if err != nil {
		return err
	}
	return nil
}

func (b *BloomFilter) GetMurmur3BitMap(key string) (int64, error) {
	murmur3, err := b.GetMurmur3(key)
	if err != nil {
		return 0, err
	}
	murmur3 = murmur3 % (1 << 32) // 取模 2^32
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 获取redis的bitmap
	res, err := b.cmd.GetBit(ctx, b.bitmapKey, int64(murmur3)).Result()
	if err != nil {
		return res, err
	}
	return res, nil
}

func (b *BloomFilter) GetMD5BitMap(key string) (int64, error) {
	md5 := b.GetMD5(key) % (1 << 32)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 获取redis的bitmap
	res, err := b.cmd.GetBit(ctx, b.bitmapKey, int64(md5)).Result()
	if err != nil {
		return res, err
	}
	return res, nil
}

func (b *BloomFilter) GetBLAKE2BitMap(key string) (int64, error) {
	blake2, err := b.GetBLAKE2(key)
	if err != nil {
		return 0, err
	}
	blake2 = blake2 % (1 << 32)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 获取redis的bitmap
	res, err := b.cmd.GetBit(ctx, b.bitmapKey, int64(blake2)).Result()
	if err != nil {
		return res, err
	}
	return res, nil
}
