package main
import "golang.org/x/crypto/bcrypt"
func GenPasswordHash(password string) (string, error){
	passwordBytes := []byte(password)
	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func checkPassword(password string, hash string) bool {
	hashBytes := []byte(hash)
	passwordBytes := []byte(password)
	err := bcrypt.CompareHashAndPassword(hashBytes, passwordBytes)
	return err == nil
}
