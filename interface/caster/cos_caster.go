package caster

import (
	"forge/biz/types"
	"forge/interface/def"
)

// CastGetOSSCredentialsReq2Params HTTP请求转服务层参数
func CastGetOSSCredentialsReq2Params(req *def.GetOSSCredentialsReq) *types.GetOSSCredentialsParams {
	if req == nil {
		return nil
	}

	return &types.GetOSSCredentialsParams{
		ResourcePath:    req.ResourcePath,
		DurationSeconds: req.DurationSeconds,
	}
}

// CastOSSCredentials2DTO 服务层结果转HTTP响应
func CastOSSCredentials2DTO(creds *types.OSSCredentials) *def.GetOSSCredentialsResp {
	if creds == nil {
		return nil
	}

	return &def.GetOSSCredentialsResp{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Expiration:      creds.Expiration,
		Region:          creds.Region,
		BucketName:      creds.BucketName,
		Provider:        creds.Provider,
		BaseURL:         creds.BaseURL,
		AccountID:       creds.AccountID,
	}
}
