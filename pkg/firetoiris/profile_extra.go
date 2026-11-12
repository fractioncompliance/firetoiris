package firetoiris

func additionalFields(form string) []Mapping {
	m := func(a, b int, key, kind string) Mapping { return Mapping{a, b, key, kind} }
	switch form {
	case "1099B":
		return []Mapping{m(545, 545, "NoncoveredSecurityInd", "enum:12:12"), m(546, 546, "ShortTermGainLossInd", "enum:1234:13"), m(546, 546, "LongTermGainLossInd", "enum:1234:24"), m(546, 546, "OrdinaryInd", "enum:1234:34"), m(547, 547, "GrossProceedsInd", "enum:12:1"), m(547, 547, "NetProceedsInd", "enum:12:2"), m(548, 555, "SoldOrDisposedDt", "date"), m(556, 568, "CUSIPNum", "text"), m(569, 607, "PropertyDesc", "text"), m(608, 615, "AcquiredDt", "date"), m(616, 616, "LossNotAllowedInd", "bool"), m(617, 617, "Form8949ApplicableCheckboxCd", "text"), m(618, 618, "CollectiblesInd", "bool"), m(619, 619, "FATCAFilingRequirementInd", "bool"), m(620, 620, "QOFInd", "bool")}
	case "1099LTC":
		return []Mapping{m(547, 547, "PerDiemInd", "enum:12:1"), m(547, 547, "ReimbursedAmountInd", "enum:12:2"), m(548, 556, "InsuredGrp.TIN", "text"), m(557, 596, "InsuredGrp.PersonName", "person"), m(597, 636, "InsuredGrp.MailingAddressGrp.USAddress.AddressLine1Txt", "text"), m(637, 676, "InsuredGrp.MailingAddressGrp.USAddress.CityNm", "text"), m(677, 678, "InsuredGrp.MailingAddressGrp.USAddress.StateAbbreviationCd", "text"), m(679, 687, "InsuredGrp.MailingAddressGrp.USAddress.ZIPCd", "text"), m(688, 688, "LTCInsuredChronicallyIllInd", "enum:12:1"), m(688, 688, "LTCInsuredTerminallyIllInd", "enum:12:2"), m(689, 696, "CertifiedDt", "date"), m(697, 697, "QualifiedContractInd", "bool")}
	case "1099LS":
		return []Mapping{m(546, 553, "SaleDt", "date"), m(663, 701, "AcquirerContactDetail.ContactNm", "text")}
	case "5498":
		return []Mapping{m(566, 573, "RMDDt", "date"), m(574, 575, "FMVInvestmentTypeCd", "codes")}

	case "1097BTC":
		return []Mapping{m(547, 547, "BondIssuerInd", "enum:12:1"), m(547, 547, "EntityRcvdBondDistriCreditInd", "enum:12:2"), m(556, 556, "IssuerAssignedUniqueIdCd", "text"), m(560, 598, "IssuerAssignedUniqueId", "text"), m(599, 601, "BondTypeCd", "text")}
	case "1098C":
		return []Mapping{m(546, 546, "CertifiesVehSoldToUnrltPrtyInd", "bool"), m(547, 547, "CertifiesVehicleNotTrnsfrInd", "bool"), m(548, 548, "CertifiesVehTrnsfrToNeedyInd", "bool"), m(549, 552, "VehicleModelYr", "text"), m(553, 565, "VehicleMakeNameTxt", "text"), m(566, 587, "VehicleModelNameTxt", "text"), m(588, 612, "VINOrOtherIdNum", "text"), m(613, 651, "CertifiesDetailedImprvDesc", "text"), m(652, 659, "ContributionDt", "date"), m(660, 660, "GoodsAndServicesYesInd", "enum:12:1"), m(660, 660, "GoodsAndServicesNoInd", "enum:12:2"), m(661, 661, "IntangibleReligiousBenefitsInd", "bool"), m(662, 662, "VehClaimDedUnderSpcfdAmtInd", "bool"), m(663, 669, "OdometerMileageQty", "integer"), m(723, 730, "SaleDt", "date"), m(731, 746, "GoodsAndServicesDesc", "text")}
	case "1098Q":
		out := []Mapping{m(546, 553, "AnnuityStartDt", "date"), m(554, 554, "AcceleratedStartDtInd", "bool"), m(580, 618, "PlanNameTxt", "text"), m(619, 638, "PlanNum", "text"), m(639, 647, "PlanSponsorEIN", "text")}
		for i, month := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
			out = append(out, m(555+2*i, 556+2*i, month+"DayNum", "integer"))
		}
		return out
	case "1099G":
		return []Mapping{m(547, 547, "TradeOrBusinessIncomeInd", "bool"), m(548, 551, "RefundCreditOffsetTaxYr", "text")}
	case "1099K":
		return []Mapping{m(547, 547, "PSEInd", "enum:12:1"), m(547, 547, "EPFOrOtherThirdPartyInd", "enum:12:2"), m(548, 548, "PaymentCardInd", "enum:12:1"), m(548, 548, "ThirdPartyNetworkInd", "enum:12:2"), m(549, 561, "PaymentTransactionCnt", "integer"), m(605, 608, "MerchantCategoryCd", "text")}
	case "1099OID":
		return []Mapping{m(547, 585, "CUSIPNumberOrObligationDesc", "text"), m(586, 586, "FATCAFilingRequirementInd", "bool")}
	case "1099PATR":
		return []Mapping{m(547, 547, "SpecifiedCooperativeInd", "bool")}
	case "1099Q":
		return []Mapping{m(547, 547, "TrusteeToTrusteeTransferInd", "enum:12:1"), m(547, 547, "QTPToRothIRATransferInd", "enum:12:2"), m(548, 548, "DistriFromPrivateQTPInd", "enum:123:1"), m(548, 548, "DistriFromStateQTPInd", "enum:123:2"), m(548, 548, "DistriFromCoverdellESAInd", "enum:123:3"), m(549, 549, "NotDesignatedBeneficiaryInd", "bool")}
	case "1099R":
		return []Mapping{m(545, 546, "Form1099RDistributionCd", "text"), m(547, 547, "TxblAmountNotDeterminedInd", "bool"), m(548, 548, "IRASEPSIMPLEInd", "bool"), m(549, 549, "TotalDistributionInd", "bool"), m(550, 551, "RcpntTotalDistributionPct", "percent"), m(552, 555, "DesignatedROTHAcctFirstYr", "text"), m(556, 556, "FATCAFilingRequirementInd", "bool"), m(557, 564, "PaymentDt", "date")}
	case "1099S":
		return []Mapping{m(547, 547, "TransferorRcvdConsiderationInd", "bool"), m(548, 555, "ClosingDt", "date"), m(556, 594, "AddressOrLegalDesc", "text"), m(595, 595, "TransferorForeignPersonInd", "bool")}
	case "1099SA":
		return []Mapping{m(545, 545, "Form1099SADistributionCd", "text"), m(547, 547, "MAMSAInd", "bool"), m(548, 548, "HSAInd", "bool"), m(549, 549, "ArcherMSAInd", "bool")}
	case "1099SB":
		return []Mapping{m(663, 701, "IssuerContactDetail.ContactNm", "text")}
	}
	return nil
}
