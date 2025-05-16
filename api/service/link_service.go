package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"durable-links-generator/api/apperrors"
	"durable-links-generator/api/models"
	"durable-links-generator/api/repository"
	"durable-links-generator/config"
	"durable-links-generator/utils"

	"github.com/rs/zerolog/log"
)

type LinkService interface {
	CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest) (*models.ShortLinkResponse, error)
	ParseLongDurableLink(longLink string) (models.CreateDurableLinkRequest, error)
	ResolveShortPath(ctx context.Context, rawURL string) (*models.LongLinkResponse, error)
	PrepareDurableLinkRequest(input map[string]any) (models.CreateDurableLinkRequest, error)
}

type linkService struct {
	repo repository.LinkRepository
	cfg  *config.Config
}

func NewLinkService(repo repository.LinkRepository, cfg *config.Config) *linkService {
	return &linkService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *linkService) getLongLinkFromHostAndPath(
	ctx context.Context,
	host string,
	path string,
) (*models.LongLinkResponse, error) {
	rawQueryStr, err := s.repo.GetQueryParamsByHostAndPath(ctx, host, path)
	if err != nil {
		return nil, err
	}

	longLink := fmt.Sprintf("%s://%s/%s", s.cfg.App.URLScheme, host, path)
	if rawQueryStr != "" {
		longLink += "?" + rawQueryStr
	}

	log.Debug().
		Str("path", path).
		Str("long_link", longLink).
		Msg("Link retrieved from service")

	return &models.LongLinkResponse{
		LongLink: longLink,
	}, nil
}

func (s *linkService) CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest) (*models.ShortLinkResponse, error) {
	warnings := []models.DurableLinkCreationWarning{}

	log.Debug().
		Str("params", fmt.Sprintf("%+v", params)).
		Msg("Durable link parameters")

	host, err := utils.CleanHost(params.DurableLinkInfo.Host)
	if err != nil {
		log.Error().
			Str("host", params.DurableLinkInfo.Host).
			Msg("Invalid host")
		return nil, fmt.Errorf("invalid host: %w", err)
	}

	if !utils.IsDomainAllowed(s.cfg.App.AllowedDomains, params.DurableLinkInfo.Link) {
		log.Error().
			Str("link", params.DurableLinkInfo.Link).
			Msg("Domain link not in allow list")
		return nil, apperrors.ErrDomainLinkNotAllowed
	}

	isi := params.DurableLinkInfo.IosParameters.IosAppStoreId

	if isi != "" {
		if !utils.IsNumericString(isi) {
			return nil, apperrors.ErrInvalidAppStoreID
		}
	}

	queryParams := url.Values{}
	queryParams.Add("link", params.DurableLinkInfo.Link)

	addParam := func(key, value string) {
		if value != "" {
			queryParams.Add(key, value)
		}
	}

	addParam("apn", params.DurableLinkInfo.AndroidParameters.AndroidPackageName)
	addParam("afl", params.DurableLinkInfo.AndroidParameters.AndroidFallbackLink)
	addParam("amv", params.DurableLinkInfo.AndroidParameters.AndroidMinPackageVersionCode)

	addParam("ifl", params.DurableLinkInfo.IosParameters.IosFallbackLink)
	addParam("ipfl", params.DurableLinkInfo.IosParameters.IosIpadFallbackLink)
	addParam("isi", isi)

	addParam("ofl", params.DurableLinkInfo.OtherPlatformParameters.FallbackURL)

	addParam("st", params.DurableLinkInfo.SocialMetaTagInfo.SocialTitle)
	addParam("sd", params.DurableLinkInfo.SocialMetaTagInfo.SocialDescription)

	si := params.DurableLinkInfo.SocialMetaTagInfo.SocialImageLink

	addParam("si", si)

	if si != "" {
		if !utils.IsURL(si) {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "MALFORMED_PARAM",
				WarningMessage: "Param 'si' is not a valid URL",
			})
		}
	}

	addParam("utm_source", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmSource)
	addParam("utm_medium", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmMedium)
	addParam("utm_campaign", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmCampaign)
	addParam("utm_term", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmTerm)
	addParam("utm_content", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmContent)
	pt := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Pt
	addParam("pt", pt)

	if isi == "" {
		if at := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.At; at != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'at' is not needed, since 'isi' is not specified.",
			})
		}
		if ct := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Ct; ct != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'ct' is not needed, since 'isi' is not specified.",
			})
		}
		if mt := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Mt; mt != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'mt' is not needed, since 'isi' is not specified.",
			})
		}
		if pt != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'pt' is not needed, since 'isi' is not specified.",
			})
		}
	}

	if pt == "" {
		if at := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.At; at != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'at' is not needed, since 'pt' is not specified.",
			})
		}
		if ct := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Ct; ct != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'ct' is not needed, since 'pt' is not specified.",
			})
		}
		if mt := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Mt; mt != "" {
			warnings = append(warnings, models.DurableLinkCreationWarning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: "Param 'mt' is not needed, since 'pt' is not specified.",
			})
		}
	}

	addParam("at", params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.At)
	addParam("ct", params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Ct)
	addParam("mt", params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Mt)

	shortPath := params.Suffix.Option == "SHORT"
	response, err := s.createOrGetShortLink(ctx, host, queryParams, shortPath)
	if err != nil {
		return nil, err
	}

	response.Warnings = warnings
	return response, nil
}

func (s *linkService) ParseLongDurableLink(longDurableLink string) (models.CreateDurableLinkRequest, error) {
	var req models.CreateDurableLinkRequest

	log.Debug().
		Str("long_link", longDurableLink).
		Msg("Parsing long durable link")

	u, err := url.Parse(longDurableLink)
	if err != nil {
		return req, apperrors.ErrInvalidURLFormat
	}

	if u.Host == "" {
		return req, apperrors.ErrHostInvalid
	}

	req.DurableLinkInfo.Host = u.Host

	params := u.Query()

	req.DurableLinkInfo.Link = params.Get("link")

	log.Debug().
		Str("link", req.DurableLinkInfo.Link).
		Msg("Parsed link")

	if s.cfg.App.DefaultAndroidPackageName != nil {
		req.DurableLinkInfo.AndroidParameters.AndroidPackageName = *s.cfg.App.DefaultAndroidPackageName
	}

	if apn := params.Get("apn"); apn != "" {
		req.DurableLinkInfo.AndroidParameters.AndroidPackageName = apn
	}
	if afl := params.Get("afl"); afl != "" {
		req.DurableLinkInfo.AndroidParameters.AndroidFallbackLink = afl
	}
	if apv := params.Get("amv"); apv != "" {
		req.DurableLinkInfo.AndroidParameters.AndroidMinPackageVersionCode = apv
	}

	if s.cfg.App.DefaultIosStoreId != nil {
		req.DurableLinkInfo.IosParameters.IosAppStoreId = *s.cfg.App.DefaultIosStoreId
	}

	if isi := params.Get("isi"); isi != "" {
		req.DurableLinkInfo.IosParameters.IosAppStoreId = isi
	}
	if ifl := params.Get("ifl"); ifl != "" {
		req.DurableLinkInfo.IosParameters.IosFallbackLink = ifl
	}
	if iflIpad := params.Get("ipfl"); iflIpad != "" {
		req.DurableLinkInfo.IosParameters.IosIpadFallbackLink = iflIpad
	}

	if ofl := params.Get("ofl"); ofl != "" {
		req.DurableLinkInfo.OtherPlatformParameters.FallbackURL = ofl
	}

	if utmSource := params.Get("utm_source"); utmSource != "" {
		req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmSource = utmSource
	}
	if utmMedium := params.Get("utm_medium"); utmMedium != "" {
		req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmMedium = utmMedium
	}
	if utmCampaign := params.Get("utm_campaign"); utmCampaign != "" {
		req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmCampaign = utmCampaign
	}
	if utmTerm := params.Get("utm_term"); utmTerm != "" {
		req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmTerm = utmTerm
	}
	if utmContent := params.Get("utm_content"); utmContent != "" {
		req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmContent = utmContent
	}
	if at := params.Get("at"); at != "" {
		req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.At = at
	}
	if ct := params.Get("ct"); ct != "" {
		req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Ct = ct
	}
	if mt := params.Get("mt"); mt != "" {
		req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Mt = mt
	}
	if pt := params.Get("pt"); pt != "" {
		req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Pt = pt
	}

	if socialTitle := params.Get("st"); socialTitle != "" {
		req.DurableLinkInfo.SocialMetaTagInfo.SocialTitle = socialTitle
	}
	if socialDescription := params.Get("sd"); socialDescription != "" {
		req.DurableLinkInfo.SocialMetaTagInfo.SocialDescription = socialDescription
	}
	if socialImageLink := params.Get("si"); socialImageLink != "" {
		req.DurableLinkInfo.SocialMetaTagInfo.SocialImageLink = socialImageLink
	}

	if pathOption := params.Get("path"); pathOption != "" {
		req.Suffix.Option = pathOption
	}

	log.Debug().
		Str("req", fmt.Sprintf("%+v", req)).
		Msg("Parsed long durable link")

	return req, nil
}

func (s *linkService) createOrGetShortLink(
	ctx context.Context,
	host string,
	queryParams url.Values,
	shortPath bool,
) (*models.ShortLinkResponse, error) {
	rawQS := queryParams.Encode()
	if shortPath {
		if path, err := s.findExistingShortLink(ctx, host, rawQS); err == nil {
			full := fmt.Sprintf("%s://%s/%s", s.cfg.App.URLScheme, host, path)
			log.Debug().
				Str("path", path).
				Str("query_params", rawQS).
				Msg("Re‑using existing short link")
			return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.DurableLinkCreationWarning{}}, nil

		} else if err != sql.ErrNoRows {
			log.Error().
				Err(err).
				Msg("Error querying for existing short link")
			return nil, err
		}
	}

	length := s.cfg.App.ShortPathLength
	if !shortPath {
		length = s.cfg.App.UnguessablePathLength
	}
	path := utils.GenerateRandomAlphanumericString(length)

	if err := s.createShortLink(ctx, host, path, rawQS, !shortPath); err != nil {
		return nil, fmt.Errorf("failed to store link: %w", err)
	}

	full := fmt.Sprintf("%s://%s/%s", s.cfg.App.URLScheme, host, path)
	log.Debug().
		Str("path", path).
		Str("query_params", rawQS).
		Msg("New link stored in database")

	return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.DurableLinkCreationWarning{}}, nil
}

func (s *linkService) findExistingShortLink(
	ctx context.Context,
	host, rawQS string,
) (string, error) {
	return s.repo.FindExistingShortLink(ctx, host, rawQS)
}

func (s *linkService) createShortLink(
	ctx context.Context,
	host, path, rawQS string,
	unguessable bool,
) error {
	return s.repo.CreateShortLink(ctx, host, path, rawQS, unguessable)
}

func (s *linkService) ResolveShortPath(ctx context.Context, rawURL string) (*models.LongLinkResponse, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, apperrors.ErrInvalidRequestedLink
	}

	normalizedHost := removePreviewFromHost(u.Host)

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 1 {
		return nil, fmt.Errorf("unexpected path format: %w", apperrors.ErrInvalidPathFormat)
	}

	return s.getLongLinkFromHostAndPath(ctx, normalizedHost, pathParts[0])
}

func removePreviewFromHost(host string) string {
	if strings.HasPrefix(host, "preview.") {
		return strings.TrimPrefix(host, "preview.")
	}
	parts := strings.SplitN(host, ".", 2) // ["acme-preview", "short.link"]
	if len(parts) == 2 && strings.HasSuffix(parts[0], "-preview") {
		app := strings.TrimSuffix(parts[0], "-preview")
		return app + "." + parts[1]
	}

	return host
}

func (s *linkService) PrepareDurableLinkRequest(input map[string]any) (models.CreateDurableLinkRequest, error) {
	var req models.CreateDurableLinkRequest

	if longLink, ok := input["longDurableLink"].(string); ok && longLink != "" {
		parsedReq, err := s.ParseLongDurableLink(longLink)
		if err != nil {
			return models.CreateDurableLinkRequest{}, err
		}
		req = parsedReq
	} else {
		reqBytes, err := json.Marshal(input)
		if err != nil {
			return models.CreateDurableLinkRequest{}, apperrors.ErrInvalidFormat
		}
		if err := json.Unmarshal(reqBytes, &req); err != nil {
			return models.CreateDurableLinkRequest{}, apperrors.ErrInvalidFormat
		}
	}

	if req.DurableLinkInfo.Host == "" {
		return models.CreateDurableLinkRequest{}, apperrors.ErrMissingHost
	}
	if req.DurableLinkInfo.Link == "" {
		return models.CreateDurableLinkRequest{}, apperrors.ErrMissingLink
	}
	if err := utils.ValidateURLScheme(req.DurableLinkInfo.Link); err != nil {
		return models.CreateDurableLinkRequest{}, err
	}

	return req, nil
}
