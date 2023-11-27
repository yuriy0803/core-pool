package proxy

import (
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/frkhash/go-frkhash"

	"github.com/frkhash/core-pool/util"
)

var (
	maxUint256                             = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	ecip1099FBlockClassic uint64           = 11700000 // classic mainnet
	ecip1099FBlockMordor  uint64           = 2520000  // mordor
	uip1FEpoch            uint64           = 22       // ubiq mainnet
	xip5Block             uint64           = 0        // expanse rebirth network
	hasher                *frkhash.Frkhash = nil
)

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string, stratum bool) (bool, bool) {
	if hasher == nil {
		if s.config.Network == "expanse" || s.config.Network == "rebirth" {
			hasher = frkhash.New(nil, nil, &xip5Block)
		} else if s.config.Network == "classic" {
			hasher = frkhash.New(&ecip1099FBlockClassic, nil, nil)
		} else if s.config.Network == "mordor" {
			hasher = frkhash.New(&ecip1099FBlockMordor, nil, nil)
		} else if s.config.Network == "ubiq" {
			hasher = frkhash.New(nil, &uip1FEpoch, nil)
		} else if s.config.Network == "ethereum" || s.config.Network == "ropsten" {
			hasher = frkhash.New(nil, nil, nil)
		} else {
			// unknown network
			log.Printf("Unknown network configuration %s", s.config.Network)
			return false, false
		}
	}
	nonceHex := params[0]
	hashNoNonce := params[1]
	mixDigest := params[2]
	nonce, _ := strconv.ParseUint(strings.Replace(nonceHex, "0x", "", -1), 16, 64)
	shareDiff := s.config.Proxy.Difficulty

	var result common.Hash
	if stratum {
		hashNoNonceTmp := common.HexToHash(params[2])

		mixDigestTmp, hashTmp := hasher.Compute(t.Height, hashNoNonceTmp, nonce)
		params[1] = hashNoNonceTmp.Hex()
		params[2] = mixDigestTmp.Hex()
		hashNoNonce = params[1]
		result = hashTmp
	} else {
		hashNoNonceTmp := common.HexToHash(hashNoNonce)
		mixDigestTmp, hashTmp := hasher.Compute(t.Height, hashNoNonceTmp, nonce)

		// check mixDigest
		if mixDigestTmp.Hex() != mixDigest {
			return false, false
		}
		result = hashTmp
	}

	// Block "difficulty" is BigInt
	// NiceHash "difficulty" is float64 ...
	// diffFloat => target; then: diffInt = 2^256 / target
	shareDiffCalc := util.TargetHexToDiff(result.Hex()).Int64()
	shareDiffFloat := util.DiffIntToFloat(shareDiffCalc)
	if shareDiffFloat < 0.0001 {
		log.Printf("share difficulty too low, %f < %d, from %v@%v", shareDiffFloat, t.Difficulty, login, ip)
		return false, false
	}

	h, ok := t.headers[hashNoNonce]
	if !ok {
		log.Printf("Stale share from %v@%v", login, ip)
		return false, false
	}

	if s.config.Proxy.Debug {
		log.Printf("Difficulty pool/block/share = %d / %d / %d(%f) from %v@%v", shareDiff, t.Difficulty, shareDiffCalc, shareDiffFloat, login, ip)
	}

	// check share difficulty
	shareTarget := new(big.Int).Div(maxUint256, big.NewInt(shareDiff))
	if result.Big().Cmp(shareTarget) > 0 {
		return false, false
	}

	// check target difficulty
	target := new(big.Int).Div(maxUint256, big.NewInt(h.diff.Int64()))
	if result.Big().Cmp(target) <= 0 {
		ok, err := s.rpc().SubmitBlock(params)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", h.height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", h.height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params, shareDiff, h.diff.Int64(), h.height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend", h.height)
			}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, h.height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params, shareDiff, h.height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
