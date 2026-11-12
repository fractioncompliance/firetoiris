package firetoiris

import "strings"

type Profile struct {
	Code    string            `json:"fire_code"`
	Form    string            `json:"form"`
	Amounts map[string]string `json:"amount_fields"`
	Fields  []Mapping         `json:"fields"`
}

type Mapping struct {
	First  int    `json:"first"`
	Last   int    `json:"last"`
	Target string `json:"target"`
	Format string `json:"format"`
}

func profile(code, form, amounts string) Profile {
	p := Profile{Code: code, Form: form, Amounts: map[string]string{}, Fields: []Mapping{}}
	for _, pair := range strings.Fields(amounts) {
		k, v, _ := strings.Cut(pair, ":")
		p.Amounts[k] = v
	}
	return p
}

func Profiles() []Profile {
	out := []Profile{
		profile("NE", "1099NEC", "1:NonemployeeCompensationAmt 4:FederalIncomeTaxWithheldAmt"),
		profile("BT", "1097BTC", "1:TotalCreditsAmt 2:JanuaryAmt 3:FebruaryAmt 4:MarchAmt 5:AprilAmt 6:MayAmt 7:JuneAmt 8:JulyAmt 9:AugustAmt A:SeptemberAmt B:OctoberAmt C:NovemberAmt D:DecemberAmt"),
		profile("3", "1098", "1:MortgageInterestReceivedAmt 2:PrinResPurchasePointsPaidAmt 3:OverpaidInterestRefundAmt 4:MortgageInsurancePremiumsAmt 6:OutstandingMortgPrincipalAmt"),
		profile("X", "1098C", "4:GrossProceedsFromSaleOfVehAmt 6:ValueOfGoodsAndSrvcForVehAmt"),
		profile("2", "1098E", "1:StudentLoanIntRcvdByLenderAmt"),
		profile("FP", "1098F", "1:TotalAmt 2:ViolationOrPotViolationAmt 3:RestitutionOrRemediationAmt 4:ComplianceAmt"),
		profile("QL", "1098Q", "1:JanuaryAmt 2:FebruaryAmt 3:MarchAmt 4:AprilAmt 5:MayAmt 6:JuneAmt 7:JulyAmt 8:AugustAmt 9:SeptemberAmt A:OctoberAmt B:NovemberAmt C:DecemberAmt D:TotalPremiumsAmt E:AnnuityStartDtAmt F:QLACFMVAmt"),
		profile("8", "1098T", "1:PaymentsReceivedQlfyExpnssAmt 3:PriorYearAdjustmentsAmt 4:GrantsOrScholarshipsAmt 5:PYAdjGrantsOrScholarshipsAmt 7:InsReimbursementOrRefundAmt"),
		profile("4", "1099A", "2:PrincipalBalanceDueAmt 4:FairMarketValueAmt"),
		profile("B", "1099B", "2:ProceedsAmt 3:CostOrOtherBasisAmt 4:FederalIncomeTaxWithheldAmt 5:NondeductibleWashSaleLossAmt 7:BarterAmt 9:TYClosedContractProfitLossAmt A:PriorYrOpenCntrctProfitLossAmt B:CYOpenCntrctProfitLossAmt C:CntrctAggregateProfitLossAmt D:AccruedMarketDiscountAmt"),
		profile("5", "1099C", "2:DebtDischargeAmt 3:InterestAmt 7:FairMarketValueAmt"),
		profile("P", "1099CAP", "2:AggregateReceivedAmt"),
		profile("1", "1099DIV", "1:TotalOrdinaryDividendsAmt 2:QualifiedDividendsAmt 3:TotalCapitalDistributionsAmt 5:Section199ADividendsAmt 6:UnrecapturedSection1250GainAmt 7:CapitalGainSect1202Amt 8:Collectibles28PercentGainAmt 9:NondividendDistributionsAmt A:FederalIncomeTaxWithheldAmt B:InvestmentExpenseAmt C:ForeignTaxesPaidAmt D:CashLiquidationDistriAmt E:NoncashLiquidationDistriAmt F:ExemptInterestDividendsAmt G:PrivateActivityBondIntDivAmt H:Section897OrdinaryDividendsAmt J:Section897CapitalGainAmt"),
		profile("F", "1099G", "1:UnemploymentCompAmt 2:StateLclRefundCreditOffsetAmt 4:FederalIncomeTaxWithheldAmt 5:RTAAPaymentsAmt 6:TaxableGrantsAmt 7:AgriculturePaymentsAmt 9:MarketGainAmt"),
		profile("6", "1099INT", "1:InterestIncomeAmt 2:EarlyWithdrawalPenaltyAmt 3:USSavingsBondsTreasObligIntAmt 4:FederalIncomeTaxWithheldAmt 5:InvestmentExpenseAmt 6:ForeignTaxesPaidAmt 8:TaxExemptInterestAmt 9:SpcfdPrvtActyBondInterestAmt A:MarketDiscountAmt B:BondPremiumAmt D:TaxExemptBondPremiumAmt E:TreasuryObligBondPremiumAmt"),
		profile("MC", "1099K", "1:GrossAmt 2:CardNotPresentTransAmt 4:FederalIncomeTaxWithheldAmt 5:JanuaryAmt 6:FebruaryAmt 7:MarchAmt 8:AprilAmt 9:MayAmt A:JuneAmt B:JulyAmt C:AugustAmt D:SeptemberAmt E:OctoberAmt F:NovemberAmt G:DecemberAmt"),
		profile("LS", "1099LS", "1:PaidToRecipientAmt"),
		profile("T", "1099LTC", "1:GrossLTCBenefitPaidAmt 2:AcceleratedDeathBenefitPaidAmt"),
		profile("A", "1099MISC", "1:RentAmt 2:RoyaltyAmt 3:OtherIncomeAmt 4:FederalIncomeTaxWithheldAmt 5:FishingBoatProceedsAmt 6:MedicalHealthCarePaymentsAmt 8:SubstitutePaymentsAmt A:CropInsuranceProceedsAmt C:AttorneyGrossProceedsPaidAmt D:Section409ADeferralsAmt E:NonqlfyDeferredCompensationAmt F:FishPurchasedForResaleAmt"),
		profile("D", "1099OID", "1:OriginalIssueDiscountAmt 2:OtherPeriodicInterestAmt 3:EarlyWithdrawalPenaltyAmt 4:FederalIncomeTaxWithheldAmt 5:BondPremiumAmt 6:TreasuryObligationOIDAmt 7:InvestmentExpenseAmt A:MarketDiscountAmt B:AcquisitionPremiumAmt C:TaxExemptOIDAmt"),
		profile("7", "1099PATR", "1:PatronageDividendsAmt 2:NonpatronageDistributionsAmt 3:PerUnitRetainAllocationsAmt 4:FederalIncomeTaxWithheldAmt 5:RedeemedNonqualifiedNoticesAmt 6:Section199AgDeductionAmt 7:InvestmentCreditAmt 8:WorkOpportunityCreditAmt B:Section199Ab7QualifiedPymtAmt C:Section199AaQualifiedItemsAmt D:Section199AaSSTBItemsAmt"),
		profile("Q", "1099Q", "1:GrossDistributionAmt 2:EarningsAmt 3:BasisAmt"),
		profile("9", "1099R", "1:GrossDistributionAmt 2:TaxableAmt 3:CapitalGainAmt 4:FederalIncomeTaxWithheldAmt 5:EmployeeContributionsAmt 6:NetUnrlzdSecuritiesApprcnAmt 8:OtherDistributionAmt 9:TotalEmployeeContributionsAmt B:IRRAllocatedAmt"),
		profile("S", "1099S", "2:TotalGrossProceedsAmt 5:BuyerRealEstateTaxAmt"),
		profile("M", "1099SA", "1:GrossDistributionAmt 2:ExcessContributionsEarningsAmt 4:FMVOnDateOfDeathAmt"),
		profile("SB", "1099SB", "1:InvestmentInContractAmt 2:SurrenderAmt"),
		profile("N", "3921", "3:ExercisePricePerShareAmt 4:FMVPerShareOnExerciseDtAmt"),
		profile("Z", "3922", "3:FMVPerShareOnGrantDtAmt 4:FMVPerShareOnExerciseDtAmt 5:ExercisePricePerShareAmt 8:ExercisePricePerShrGrantDtAmt"),
		profile("L", "5498", "1:IRAContributionsAmt 2:RolloverContributionAmt 3:ConversionToRothIRATaxbleAmt 4:RecharacterizedContributionAmt 5:EOYFMVAmt 6:LifeInsIncludedInIRAContriAmt 7:FairMarketValueAmt 8:SEPContributionsAmt 9:SIMPLEContributionsAmt A:RothIRAContributionsAmt B:RMDAmt C:PostponedOrLateContributionGrp.ContributionAmt D:DistributionRepaymentAmt"),
		profile("V", "5498ESA", "1:CoverdellESAContributionAmt 2:RolloverContributionAmt"),
		profile("K", "5498SA", "1:ArcherMSAContributionAmt 2:TotalContributionsAmt 3:PriorYearContributionAmt 4:RolloverContributionAmt 5:EOYFairMarketValueAmt"),
		profile("W", "W2G", "1:GamblingReportableWinningAmt 2:FederalIncomeTaxWithheldAmt 7:GamblingWinFromIdntclWagersAmt"),
	}
	for i := range out {
		out[i].Fields = append(tailFields(out[i].Form), additionalFields(out[i].Form)...)
	}
	return out
}

func tailFields(form string) []Mapping {
	m := func(a, b int, key, kind string) Mapping { return Mapping{a, b, key, kind} }
	switch form {
	case "1099NEC":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice"), m(547, 547, "DirectSaleAboveThresholdInd", "bool")}
	case "1099MISC":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice"), m(547, 547, "DirectSaleAboveThresholdInd", "bool"), m(548, 548, "FATCAFilingRequirementInd", "bool")}
	case "1098E":
		return []Mapping{m(547, 547, "NotLoanOrigFeeOrCapInterestInd", "bool")}
	case "1098T":
		return []Mapping{m(544, 544, "StudentTINSolicitationInd", "bool"), m(547, 547, "HalfTimeStudentInd", "bool"), m(548, 548, "GraduateStudentInd", "bool"), m(549, 549, "PymtRcvdIncldAmtSpcfdPeriodInd", "bool")}
	case "1098":
		return []Mapping{m(544, 551, "MortgageOriginationDt", "date"), m(552, 552, "PropAddrSameBorrowerAddrInd", "bool"), m(553, 591, "PropertyDesc", "text"), m(592, 630, "OtherTxt", "text"), m(670, 673, "PropertiesSecuringMortgageCnt", "integer"), m(723, 730, "MortgageAcquisitionDt", "date")}
	case "1098F":
		return []Mapping{m(544, 551, "SuitOrderOrAgreementDt", "date"), m(552, 590, "CourtOrEntityNm", "text"), m(591, 629, "CaseNum", "text"), m(630, 668, "CaseNameOrPartiesNamesSOATxt", "text"), m(669, 673, "Form1098FCd", "codes")}
	case "1099A":
		return []Mapping{m(547, 547, "PersonallyLiableInd", "bool"), m(548, 555, "AbandonmentOrAcquisitionDt", "date"), m(556, 594, "PropertyDesc", "text")}
	case "1099C":
		return []Mapping{m(547, 547, "IdentifiableEventCd", "text"), m(548, 555, "IdentifiableEventDt", "date"), m(556, 594, "DebtDesc", "text"), m(595, 595, "DebtorLiableDebtRepaymentInd", "bool")}
	case "1099CAP":
		return []Mapping{m(548, 555, "SaleOrExchangeDt", "date"), m(608, 615, "SharesExchangedCnt", "integer"), m(616, 625, "StockClassDesc", "text")}
	case "1099DIV":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice"), m(587, 587, "FATCAFilingRequirementInd", "bool")}
	case "1099INT":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice"), m(587, 599, "TaxExemptTaxCreditBondCUSIPNum", "text"), m(600, 600, "FATCAFilingRequirementInd", "bool")}
	case "1099OID":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice")}
	case "1099G", "1099B", "1099K", "1099PATR":
		return []Mapping{m(544, 544, "SecondTINNoticeInd", "notice")}
	case "3921":
		return []Mapping{m(547, 554, "StockOptionGrantedDt", "date"), m(555, 562, "StockOptionExercisedDt", "date"), m(563, 570, "SharesTransferredCnt", "integer")}
	case "3922":
		return []Mapping{m(547, 554, "StockOptionGrantedDt", "date"), m(555, 562, "StockOptionExercisedDt", "date"), m(563, 570, "SharesTransferredCnt", "integer"), m(571, 578, "LegalTitleTransferDt", "date")}
	case "5498SA":
		return []Mapping{m(547, 547, "MAMSAInd", "bool"), m(548, 548, "HSAInd", "bool"), m(549, 549, "ArcherMSAInd", "bool")}
	case "5498":
		return []Mapping{m(547, 547, "IRAInd", "bool"), m(548, 548, "SEPInd", "bool"), m(549, 549, "SIMPLEInd", "bool"), m(550, 550, "RothIRAInd", "bool"), m(551, 551, "RMDInd", "bool"), m(552, 555, "PostponedOrLateContributionGrp.ContributionYr", "text"), m(556, 557, "PostponedOrLateContributionGrp.PostponedOrLateContributionCd", "text"), m(558, 563, "PostponedOrLateContributionGrp.PostponedOrLateContriReasonTxt", "text"), m(564, 565, "DistributionRepaymentCd", "text")}
	case "W2G":
		return []Mapping{m(547, 547, "GamblingWinWagerTypeCd", "text"), m(548, 555, "GamblingWinningDt", "date"), m(556, 570, "GamblingWinningTransactionDesc", "text"), m(571, 575, "GamblingWinningEventDesc", "text"), m(576, 580, "GamblingWinCashierId", "text"), m(581, 585, "GamblingWinWindowNum", "text"), m(586, 600, "RecipientFirstAdditionalIdNum", "text"), m(601, 615, "RecipientSecondAdditionalIdNum", "text")}
	}
	return []Mapping{}
}
