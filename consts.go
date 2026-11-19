package gomium

import (
	cef "github.com/timzifer/gomium/internal/cef"
)

// Constants of the binding, re-exported unchanged.

// LevelTrace is the slog level below slog.LevelDebug that SetMemoryLogger,
// SetDebugLogger and the noisiest app steps log at.
const LevelTrace = cef.LevelTrace

// CertificateAccept continues the load despite the invalid certificate.
const CertificateAccept = cef.CertificateAccept

// CertificateDecideAsync defers to a later callback.Continue call.
const CertificateDecideAsync = cef.CertificateDecideAsync

// CertificateDeny cancels the request (the CEF default).
const CertificateDeny = cef.CertificateDeny

const CmEditCanCopy = cef.CmEditCanCopy

const CmEditCanCut = cef.CmEditCanCut

const CmEditCanDelete = cef.CmEditCanDelete

const CmEditCanPaste = cef.CmEditCanPaste

const CmEditCanRedo = cef.CmEditCanRedo

const CmEditCanSelectAll = cef.CmEditCanSelectAll

// EditFlags - which edit operations apply (bitmask).
const CmEditCanUndo = cef.CmEditCanUndo

const CmTypeEditable = cef.CmTypeEditable

const CmTypeFrame = cef.CmTypeFrame

const CmTypeLink = cef.CmTypeLink

const CmTypeMedia = cef.CmTypeMedia

// TypeFlags - what the click hit (bitmask).
const CmTypeNone = cef.CmTypeNone

const CmTypePage = cef.CmTypePage

const CmTypeSelection = cef.CmTypeSelection

const CookiePriorityHigh = cef.CookiePriorityHigh

const CookiePriorityLow = cef.CookiePriorityLow

const CookiePriorityMedium = cef.CookiePriorityMedium

const CookieSameSiteLaxMode = cef.CookieSameSiteLaxMode

const CookieSameSiteNoRestriction = cef.CookieSameSiteNoRestriction

const CookieSameSiteStrictMode = cef.CookieSameSiteStrictMode

const CookieSameSiteUnspecified = cef.CookieSameSiteUnspecified

// CursorType values, explicitly mapped to cef_cursor_type_t so inserted enum
// values in future CEF versions cannot silently shift the constants.
const (
	CursorTypeALIAS                    = cef.CursorTypeALIAS
	CursorTypeCELL                     = cef.CursorTypeCELL
	CursorTypeCOLUMNRESIZE             = cef.CursorTypeCOLUMNRESIZE
	CursorTypeCONTEXTMENU              = cef.CursorTypeCONTEXTMENU
	CursorTypeCOPY                     = cef.CursorTypeCOPY
	CursorTypeCROSS                    = cef.CursorTypeCROSS
	CursorTypeCUSTOM                   = cef.CursorTypeCUSTOM
	CursorTypeEASTPANNING              = cef.CursorTypeEASTPANNING
	CursorTypeEASTRESIZE               = cef.CursorTypeEASTRESIZE
	CursorTypeEASTWESTRESIZE           = cef.CursorTypeEASTWESTRESIZE
	CursorTypeGRAB                     = cef.CursorTypeGRAB
	CursorTypeGRABBING                 = cef.CursorTypeGRABBING
	CursorTypeHAND                     = cef.CursorTypeHAND
	CursorTypeHELP                     = cef.CursorTypeHELP
	CursorTypeIBEAM                    = cef.CursorTypeIBEAM
	CursorTypeMIDDLEPANNING            = cef.CursorTypeMIDDLEPANNING
	CursorTypeMOVE                     = cef.CursorTypeMOVE
	CursorTypeNODROP                   = cef.CursorTypeNODROP
	CursorTypeNONE                     = cef.CursorTypeNONE
	CursorTypeNORTHEASTPANNING         = cef.CursorTypeNORTHEASTPANNING
	CursorTypeNORTHEASTRESIZE          = cef.CursorTypeNORTHEASTRESIZE
	CursorTypeNORTHEASTSOUTHWESTRESIZE = cef.CursorTypeNORTHEASTSOUTHWESTRESIZE
	CursorTypeNORTHPANNING             = cef.CursorTypeNORTHPANNING
	CursorTypeNORTHRESIZE              = cef.CursorTypeNORTHRESIZE
	CursorTypeNORTHSOUTHRESIZE         = cef.CursorTypeNORTHSOUTHRESIZE
	CursorTypeNORTHWESTPANNING         = cef.CursorTypeNORTHWESTPANNING
	CursorTypeNORTHWESTRESIZE          = cef.CursorTypeNORTHWESTRESIZE
	CursorTypeNORTHWESTSOUTHEASTRESIZE = cef.CursorTypeNORTHWESTSOUTHEASTRESIZE
	CursorTypeNOTALLOWED               = cef.CursorTypeNOTALLOWED
	CursorTypePOINTER                  = cef.CursorTypePOINTER
	CursorTypePROGRESS                 = cef.CursorTypePROGRESS
	CursorTypeROWRESIZE                = cef.CursorTypeROWRESIZE
	CursorTypeSOUTHEASTPANNING         = cef.CursorTypeSOUTHEASTPANNING
	CursorTypeSOUTHEASTRESIZE          = cef.CursorTypeSOUTHEASTRESIZE
	CursorTypeSOUTHPANNING             = cef.CursorTypeSOUTHPANNING
	CursorTypeSOUTHRESIZE              = cef.CursorTypeSOUTHRESIZE
	CursorTypeSOUTHWESTPANNING         = cef.CursorTypeSOUTHWESTPANNING
	CursorTypeSOUTHWESTRESIZE          = cef.CursorTypeSOUTHWESTRESIZE
	CursorTypeVERTICALTEXT             = cef.CursorTypeVERTICALTEXT
	CursorTypeWAIT                     = cef.CursorTypeWAIT
	CursorTypeWESTPANNING              = cef.CursorTypeWESTPANNING
	CursorTypeWESTRESIZE               = cef.CursorTypeWESTRESIZE
	CursorTypeZOOMIN                   = cef.CursorTypeZOOMIN
	CursorTypeZOOMOUT                  = cef.CursorTypeZOOMOUT
)

// JsDialogDefault leaves the dialog to CEF's default implementation.
const JsDialogDefault = cef.JsDialogDefault

// JsDialogHandled: the handler shows an own dialog and will execute the
// callback once it is dismissed.
const JsDialogHandled = cef.JsDialogHandled

// JsDialogSuppressed drops the dialog silently. Preferable to an
// immediate callback - CEF uses it to detect alert spamming.
const JsDialogSuppressed = cef.JsDialogSuppressed

const AuthDefault = cef.AuthDefault

const AuthHandled = cef.AuthHandled

const FocusSourceNavigation = cef.FocusSourceNavigation

const FocusSourceSystem = cef.FocusSourceSystem

const FileDialogOpen = cef.FileDialogOpen

const FileDialogOpenMultiple = cef.FileDialogOpenMultiple

const FileDialogOpenFolder = cef.FileDialogOpenFolder

const FileDialogSave = cef.FileDialogSave

const JsDialogTypeAlert = cef.JsDialogTypeAlert

const JsDialogTypeConfirm = cef.JsDialogTypeConfirm

const JsDialogTypePrompt = cef.JsDialogTypePrompt

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierAlt = cef.KeyEventModifierAlt

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierCapsLock = cef.KeyEventModifierCapsLock

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierCommand = cef.KeyEventModifierCommand

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierControl = cef.KeyEventModifierControl

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierIsKeyPad = cef.KeyEventModifierIsKeyPad

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierIsLeft = cef.KeyEventModifierIsLeft

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierIsRight = cef.KeyEventModifierIsRight

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierLeftMouseButton = cef.KeyEventModifierLeftMouseButton

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierMiddleMouseButton = cef.KeyEventModifierMiddleMouseButton

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierNone = cef.KeyEventModifierNone

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierNumLockOn = cef.KeyEventModifierNumLockOn

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierRightMouseButton = cef.KeyEventModifierRightMouseButton

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const KeyEventModifierShift = cef.KeyEventModifierShift

const KeyEventTypeChar = cef.KeyEventTypeChar

const KeyEventTypeKeyDown = cef.KeyEventTypeKeyDown

const KeyEventTypeKeyUp = cef.KeyEventTypeKeyUp

const KeyEventTypeRawKeyDown = cef.KeyEventTypeRawKeyDown

const LOGSEVERITY_DEFAULT = cef.LOGSEVERITY_DEFAULT

const LOGSEVERITY_DISABLE = cef.LOGSEVERITY_DISABLE

const LOGSEVERITY_ERROR = cef.LOGSEVERITY_ERROR

const LOGSEVERITY_INFO = cef.LOGSEVERITY_INFO

const LOGSEVERITY_VERBOSE = cef.LOGSEVERITY_VERBOSE

const LOGSEVERITY_WARNING = cef.LOGSEVERITY_WARNING

// Command ids of Chromium's built-in menu entries, usable with
// ContextMenuCallback.Continue.
const MenuIdBack = cef.MenuIdBack

const MenuIdCopy = cef.MenuIdCopy

const MenuIdCut = cef.MenuIdCut

const MenuIdDelete = cef.MenuIdDelete

const MenuIdFind = cef.MenuIdFind

const MenuIdForward = cef.MenuIdForward

const MenuIdPaste = cef.MenuIdPaste

const MenuIdPrint = cef.MenuIdPrint

const MenuIdRedo = cef.MenuIdRedo

const MenuIdReload = cef.MenuIdReload

const MenuIdSelectAll = cef.MenuIdSelectAll

const MenuIdUndo = cef.MenuIdUndo

const MenuIdViewSource = cef.MenuIdViewSource

const MenuItemTypeCheck = cef.MenuItemTypeCheck

const MenuItemTypeCommand = cef.MenuItemTypeCommand

const MenuItemTypeRadio = cef.MenuItemTypeRadio

const MenuItemTypeSeparator = cef.MenuItemTypeSeparator

const MenuItemTypeSubMenu = cef.MenuItemTypeSubMenu

const MouseButtonDown = cef.MouseButtonDown

const MouseButtonLeft = cef.MouseButtonLeft

const MouseButtonMiddle = cef.MouseButtonMiddle

const MouseButtonRight = cef.MouseButtonRight

const MouseButtonUp = cef.MouseButtonUp

const PidBrowser = cef.PidBrowser

const PidRenderer = cef.PidRenderer

const ResourceLoadCancel = cef.ResourceLoadCancel

const ResourceLoadContinue = cef.ResourceLoadContinue

const ResourceLoadContinueAsync = cef.ResourceLoadContinueAsync

// SchemeCorsEnabled allows CORS requests to the scheme.
const SchemeCorsEnabled = cef.SchemeCorsEnabled

// SchemeCspBypassing exempts the scheme from Content-Security-Policy.
const SchemeCspBypassing = cef.SchemeCspBypassing

// SchemeDisplayIsolated allows the scheme to be displayed only from content
// of the same scheme.
const SchemeDisplayIsolated = cef.SchemeDisplayIsolated

// SchemeFetchEnabled allows fetch() and XMLHttpRequest against the scheme.
const SchemeFetchEnabled = cef.SchemeFetchEnabled

// SchemeLocal treats the scheme like file://.
const SchemeLocal = cef.SchemeLocal

// SchemeSecure treats the scheme like https - no mixed-content warnings, and
// features that require a secure context work.
const SchemeSecure = cef.SchemeSecure

// SchemeStandard gives the scheme an origin and the usual URL parsing
// ("scheme://host/path"). Everything except SchemeLocal builds on it.
const SchemeStandard = cef.SchemeStandard

// TidFile is kept as an alias for the closest CEF 151 equivalent of the
// former TID_FILE thread. TID_DB and TID_CACHE no longer exist.
const TidFile = cef.TidFile

const TidFileBackground = cef.TidFileBackground

const TidFileUserBlocking = cef.TidFileUserBlocking

const TidFileUserVisible = cef.TidFileUserVisible

const TidIO = cef.TidIO

const TidProcessLauncher = cef.TidProcessLauncher

const TidRenderer = cef.TidRenderer

const TidUI = cef.TidUI

const ValueTypeBinary = cef.ValueTypeBinary

const ValueTypeBool = cef.ValueTypeBool

const ValueTypeDictionary = cef.ValueTypeDictionary

const ValueTypeDouble = cef.ValueTypeDouble

const ValueTypeInt = cef.ValueTypeInt

const ValueTypeInvalid = cef.ValueTypeInvalid

const ValueTypeList = cef.ValueTypeList

const ValueTypeNull = cef.ValueTypeNull

const ValueTypeString = cef.ValueTypeString

const WodCurrentTab = cef.WodCurrentTab

const WodIgnoreAction = cef.WodIgnoreAction

const WodNewBackgroundTab = cef.WodNewBackgroundTab

const WodNewForegroundTab = cef.WodNewForegroundTab

const WodNewPictureInPicture = cef.WodNewPictureInPicture

const WodNewPopup = cef.WodNewPopup

const WodNewWindow = cef.WodNewWindow

const WodOffTheRecord = cef.WodOffTheRecord

const WodSaveToDisk = cef.WodSaveToDisk

const WodSingletonTab = cef.WodSingletonTab

const WodSwitchToTab = cef.WodSwitchToTab

const WodUnknown = cef.WodUnknown
