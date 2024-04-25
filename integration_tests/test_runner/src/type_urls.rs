// Althea Msg Types

// microtx
pub const MSG_MICROTX_TYPE_URL: &str = "/althea.microtx.v1.MsgMicrotx";
pub const MSG_LIQUIFY_TYPE_URL: &str = "/althea.microtx.v1.MsgLiquify";

// authz
pub const GENERIC_AUTHORIZATION_TYPE_URL: &str = "/cosmos.authz.v1beta1.GenericAuthorization";
pub const MSG_GRANT_TYPE_URL: &str = "/cosmos.authz.v1beta1.MsgGrant";
pub const GRANT_TYPE_URL: &str = "/cosmos.authz.v1beta1.Grant";
pub const MSG_EXEC_TYPE_URL: &str = "/cosmos.authz.v1beta1.MsgExec";

// Althea Proposal Types
pub const UPGRADE_PROXY_PROPOSAL_TYPE_URL: &str = "/althea.nativedex.v1.UpgradeProxyProposal";
pub const COLLECT_TREASURY_PROPOSAL_TYPE_URL: &str = "/althea.nativedex.v1.CollectTreasuryProposal";
pub const SET_TREASURY_PROPOSAL_TYPE_URL: &str = "/althea.nativedex.v1.SetTreasuryProposal";
pub const AUTHORITY_TRANSFER_PROPOSAL_TYPE_URL: &str =
    "/althea.nativedex.v1.AuthorityTransferProposal";
pub const HOT_PATH_OPEN_PROPOSAL_TYPE_URL: &str = "/althea.nativedex.v1.HotPathOpenProposal";
pub const SET_SAFE_MODE_PROPOSAL_TYPE_URL: &str = "/althea.nativedex.v1.SetSafeModeProposal";
pub const TRANSFER_GOVERNANCE_PROPOSAL_TYPE_URL: &str =
    "/althea.nativedex.v1.TransferGovernanceProposal";

// bank msgs
pub const MSG_SEND_TYPE_URL: &str = "/cosmos.bank.v1beta1.MsgSend";
pub const MSG_MULTI_SEND_TYPE_URL: &str = "/cosmos.bank.v1beta1.MsgMultiSend";

// distribution msgs
pub const MSG_SET_WITHDRAW_ADDRESS_TYPE_URL: &str =
    "/cosmos.distribution.v1beta1.MsgSetWithdrawAddress";

// cosmos-sdk proposals
pub const PARAMETER_CHANGE_PROPOSAL_TYPE_URL: &str =
    "/cosmos.params.v1beta1.ParameterChangeProposal";
pub const SOFTWARE_UPGRADE_PROPOSAL_TYPE_URL: &str =
    "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal";

// ibc-go msgs
pub const MSG_TRANSFER_TYPE_URL: &str = "/ibc.applications.transfer.v1.MsgTransfer";

// canto
pub const MSG_CONVERT_ERC20_TYPE_URL: &str = "/canto.erc20.v1.MsgConvertERC20";
pub const MSG_CONVERT_COIN_TYPE_URL: &str = "/canto.erc20.v1.MsgConvertCoin";

pub const REGISTER_COIN_PROPOSAL_TYPE_URL: &str = "/canto.erc20.v1.RegisterCoinProposal";
pub const REGISTER_ERC20_PROPOSAL_TYPE_URL: &str = "/canto.erc20.v1.RegisterERC20Proposal";
