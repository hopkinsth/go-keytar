package keytar

/*
#cgo pkg-config: glib-2.0 gnome-keyring-1

// Standard includes
#include <stdlib.h>
#include <string.h>

// GNOME includes
#include <glib.h>
#include <gnome-keyring.h>

// TODO: Eventually it'd be nice to switch to GNOME's simple password storage,
// but manually manipulating items allows us to work with older versions of
// GNOME.  The better option would be to simply switch to another library,
// because gnome-keyring is deprecated and unreliable.

// Generates an attribute structure for creating/searching based on the account
// and service
GnomeKeyringAttributeList * createAttributes(
	const char * service,
	const char * account
) {
	// Allocate the list
	GnomeKeyringAttributeList * result = gnome_keyring_attribute_list_new();

	// Add the attributes
	gnome_keyring_attribute_list_append_string(result, "service", service);
	gnome_keyring_attribute_list_append_string(result, "account", account);

	// All done
	return result;
}

// Releases the attribute structure generated by createAttributes
void freeAttributes(GnomeKeyringAttributeList * list) {
	gnome_keyring_attribute_list_free(list);
}

// Adds a password to the default keychain.  All arguments must be UTF-8 encoded
// and null-terminated.
int addPassword(
	const char * displayName,
	const char * service,
	const char * account,
	const char * password
) {
	// Create the item attributes
	GnomeKeyringAttributeList * attributes = createAttributes(
		service,
		account
	);

	// Create the item
	guint32 item = 0;
	GnomeKeyringResult result = gnome_keyring_item_create_sync(
		NULL,
		GNOME_KEYRING_ITEM_GENERIC_SECRET,
		displayName,
		attributes,
		password,
		FALSE,
		&item
	);

	// Release attributes
	freeAttributes(attributes);

	// Check the result
	if (result != GNOME_KEYRING_RESULT_OK) {
		return -1;
	}

	// All done
	return 0;
}

// Gets a password from the default keychain.  All arguments must be UTF-8
// encoded and null-terminated.  On success, the password argument will be set
// to a null-terminated string that must be released with free.
int getPassword(const char * service, const char * account, char ** password) {
	// Create the item attributes
	GnomeKeyringAttributeList * attributes = createAttributes(
		service,
		account
	);

	// Find the item
	GList * matches = NULL;
	GnomeKeyringResult result = gnome_keyring_find_items_sync(
		GNOME_KEYRING_ITEM_GENERIC_SECRET,
		attributes,
		&matches
	);

	// Release attributes
	freeAttributes(attributes);

	// Check the results
	if (result != GNOME_KEYRING_RESULT_OK || g_list_length(matches) == 0) {
		gnome_keyring_found_list_free(matches);
		*password == NULL;
		return -1;
	}

	// Grab the first result and extract the password
	const char * secret = ((GnomeKeyringFound *)(matches->data))->secret;
	*password = malloc(strlen(secret) + 1);
	strcpy(*password, secret);

	// Free the results
	gnome_keyring_found_list_free(matches);

	// All done
	return 0;
}

// Deletes a password from the default keychain.  All arguments must be UTF-8
// encoded and null-terminated.
int deletePassword(const char * service, const char * account) {
	// Create the item attributes
	GnomeKeyringAttributeList * attributes = createAttributes(
		service,
		account
	);

	// Find the item
	GList * matches = NULL;
	GnomeKeyringResult result = gnome_keyring_find_items_sync(
		GNOME_KEYRING_ITEM_GENERIC_SECRET,
		attributes,
		&matches
	);

	// Release attributes
	freeAttributes(attributes);

	// Check the results
	if (result != GNOME_KEYRING_RESULT_OK || g_list_length(matches) == 0) {
		gnome_keyring_found_list_free(matches);
		return -1;
	}

	// Get the id of the first result
	guint item = ((GnomeKeyringFound *)(matches->data))->item_id;

	// Free the results
	gnome_keyring_found_list_free(matches);

	// Delete the item
	result = gnome_keyring_item_delete_sync(NULL, item);

	// Check the result
	if (result != GNOME_KEYRING_RESULT_OK) {
		return -1;
	}

	// All done
	return 0;
}
*/
import "C"

import (
	// System imports
	"fmt"
	"unsafe"
)

// Linux keychain implementation
type keychainLinux struct{}

func (*keychainLinux) AddPassword(service, account, password string) error {
	// Validate input
	serviceValid := isValidNonNullUTF8(service)
	accountValid := isValidNonNullUTF8(account)
	passwordValid := isValidNonNullUTF8(password)
	if !(serviceValid && accountValid && passwordValid) {
		return ErrInvalidValue
	}

	// Compute a display name and convert it to a C string
	display := fmt.Sprintf("%s@%s", service, account)

	// Convert values to C strings
	displayCStr := C.CString(display)
	defer C.free(unsafe.Pointer(displayCStr))
	serviceCStr := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCStr))
	accountCStr := C.CString(account)
	defer C.free(unsafe.Pointer(accountCStr))
	passwordCStr := C.CString(password)
	C.free(unsafe.Pointer(passwordCStr))

	// Do the add and check for errors
	if C.addPassword(displayCStr, serviceCStr, accountCStr, passwordCStr) < 0 {
		return ErrUnknown
	}

	// All done
	return nil
}

func (*keychainLinux) GetPassword(service, account string) (string, error) {
	// Validate input
	serviceValid := isValidNonNullUTF8(service)
	accountValid := isValidNonNullUTF8(account)
	if !(serviceValid && accountValid) {
		return "", ErrInvalidValue
	}

	// Convert values to C strings
	serviceCStr := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCStr))
	accountCStr := C.CString(account)
	defer C.free(unsafe.Pointer(accountCStr))

	// Get the password and check for errors
	var passwordCStr *C.char
	if C.getPassword(serviceCStr, accountCStr, &passwordCStr) < 0 {
		return "", ErrNotFound
	}

	// If there was a match, convert it and free the underlying C string
	password := C.GoString(passwordCStr)
	C.free(unsafe.Pointer(passwordCStr))

	// All done
	return password, nil
}

func (*keychainLinux) DeletePassword(service, account string) error {
	// Validate input
	serviceValid := isValidNonNullUTF8(service)
	accountValid := isValidNonNullUTF8(account)
	if !(serviceValid && accountValid) {
		return ErrInvalidValue
	}

	// Convert values to C strings
	serviceCStr := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCStr))
	accountCStr := C.CString(account)
	defer C.free(unsafe.Pointer(accountCStr))

	// Delete the password and check for errors
	if C.deletePassword(serviceCStr, accountCStr) < 0 {
		return ErrUnknown
	}

	// All done
	return nil
}

func init() {
	// Register the Linux keychain implementation if keychain services are
	// available
	if C.gnome_keyring_is_available() == C.TRUE {
		keychain = &keychainLinux{}
	}
}
