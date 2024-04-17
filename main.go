package main

import (
	"AAA/curve25519"
	"AAA/implementation/authentication"
	"AAA/implementation/bls"
	"AAA/implementation/elgamal"
	"AAA/implementation/mabe"
	"bytes"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/simple"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/util/random"
)

func testBLSSignature() {
	msg := []byte("Hello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-ShachamHello Boneh-Lynn-Shacham")
	suite := bn256.NewSuite()
	private, public := bls.NewKeyPair(suite, random.New())
	// s, _ := private.MarshalBinary()
	// fmt.Println(len(s))
	sig, err := bls.Sign(suite, private, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 0; i < 100; i++ {
		bls.Verify(suite, public, msg, sig)
	}
}

func testThresholdElGamal() {
	// now := time.Now()
	suite := curve25519.NewBlakeSHA256Curve25519(false)
	t := 5
	n := 30
	masterSecretKey, masterPublicKey := elgamal.NewKeyPair(suite, random.New())
	keyPairs := elgamal.GenerateKeyPair(suite, t, n, masterSecretKey)
	msg := []byte("hello")
	// fmt.Println(time.Since(now))
	c1, c2, err := elgamal.Encrypt(suite, masterPublicKey, msg)

	if err != nil {
		log.Println(err)
	}

	var subPoints []kyber.Point
	var xs []int

	for i := 0; i < t; i++ {
		// compute u:=secretKey*rp
		u, err := elgamal.Deal(suite, c2, keyPairs[i].SecretKey)
		if err != nil {
			log.Println(err)
			return
		}
		subPoints = append(subPoints, u)
		xs = append(xs, keyPairs[i].X)
	}
	now1 := time.Now()
	for i := 0; i < 100; i++ {
		elgamal.ThresholdDecrypt(suite, subPoints, xs, c1)
	}
	fmt.Println("descryption cost:", time.Since(now1)/100)
	msg2, err2 := elgamal.ThresholdDecrypt(suite, subPoints, xs, c1)

	if err2 != nil {
		log.Println(err2)
	}
	result := bytes.Compare(msg, msg2)
	if result == 0 {
		log.Println("true")
	} else {
		log.Println("false")
	}

}

func main() {
	ComparisonTest()
}

func test2() {
	// Instantiation of a trusted entity that
	// will generate master keys and FE key
	l := 2                  // length of input vectors
	bound := big.NewInt(10) // upper bound for input vector coordinates
	modulusLength := 2048   // bit length of prime modulus p
	trustedEnt, _ := simple.NewDDHPrecomp(l, modulusLength, bound)
	msk, mpk, _ := trustedEnt.GenerateMasterKeys()

	y := data.NewVector([]*big.Int{big.NewInt(1), big.NewInt(2)})
	feKey, _ := trustedEnt.DeriveKey(msk, y)

	// Simulate instantiation of encryptor
	// Encryptor wants to hide x and should be given
	// master public key by the trusted entity
	enc := simple.NewDDHFromParams(trustedEnt.Params)
	x := data.NewVector([]*big.Int{big.NewInt(3), big.NewInt(4)})
	cipher, _ := enc.Encrypt(x, mpk)

	// Simulate instantiation of decryptor that decrypts the cipher
	// generated by encryptor.
	dec := simple.NewDDHFromParams(trustedEnt.Params)
	// decrypt to obtain the result: inner prod of x and y
	// we expect xy to be 11 (e.g. <[1,2],[3,4]>)
	xy, _ := dec.Decrypt(cipher, feKey, y)
	print(xy)
	print("123")
}

func MABETest() {
	inst := mabe.NewMAABE()
	auditor, err := inst.InitAuditor(false)
	if err != nil {
		fmt.Println(err)
	}
	auth1, err := inst.NewMAABEAuth("auth1", []string{"America", "China", "France", "Britain"})
	if err != nil {
		fmt.Errorf("generating auth1 faces error!")
	}
	auth2, err := inst.NewMAABEAuth("auth1", []string{"Computer", "Math", "English", "Physics"})
	if err != nil {
		fmt.Errorf("generating auth1 faces error!")
	}

	pub1 := auth1.PubKeys()
	pub2 := auth2.PubKeys()

	msg := "i am tony"

	// 构造策略信息
	// "((0 AND 1) OR (2 AND 3)) AND 5",
	msp, err := mabe.BooleanToMSP("((America AND Physics) OR (China AND English) )", false)
	if err != nil {
		panic(err)
	}

	cipher, err := inst.Encrypt(msg, msp, []*mabe.MAABEPubKey{pub1, pub2})
	if err != nil {
		panic(err)
	}
	// cipherLength := 0

	decKey1, err := auth1.GenerateAttribKeys("edge server1", []string{"America", "France"})
	if err != nil {
		panic(err)
	}
	decKey2, err := auth2.GenerateAttribKeys("edge server1", []string{"Physics", "English"})

	if err != nil {
		panic(err)
	}

	msgCheck, err := inst.Decrypt(cipher, append(decKey1, decKey2...))
	if err != nil {
		panic(err)
	}

	fmt.Println(msgCheck)

	msgCheck2, err := inst.Audit(cipher, auditor.SK)
	if err != nil {
		panic(err)
	}

	fmt.Println(msgCheck2)
}

// 整个流程耗时13ms
// GenToken:1ms
// GenBlindTOken:3ms
// GenProof: 1.5ms
// verify:5ms
// u1:64字节 on G1
// Y: 128 byte on G2
// scalar 32
// te: 16

// abcd+te: 64+ 128 + 128 + 64  sig: 64, v1v2v3v4+tp:128+128+32+32+16

// access : 边缘服务器耗时8.8 ms 用户耗时4.8ms
func Authen() (time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, time.Duration) {

	// init
	suite := bn256.NewSuite()
	csk1, cpk1 := authentication.NewKeyPair(suite, random.New())
	csk2, cpk2 := authentication.NewKeyPair(suite, random.New())
	csk3, cpk3 := authentication.NewKeyPair(suite, random.New())
	ask, apk := authentication.NewKeyPair(suite, random.New())
	pp := authentication.NewPP(cpk1, cpk2, cpk3, apk)

	// user registration
	te := time.Now().UnixMicro()
	y, Y := authentication.NewKeyPair(suite, random.New())
	yb, _ := y.MarshalBinary()
	print(len(yb))
	// y1, _ := authentication.NewKeyPair(suite, random.New())
	Yb, _ := Y.MarshalBinary()
	u, _ := authentication.Hash2(suite, append(Yb, []byte(fmt.Sprintf("%d", te))...))
	H := u.Mul(y, u)
	Hb, _ := H.MarshalBinary()
	println(len(Hb))
	IDs := "service1:service2:service3:service4"
	tk, err := authentication.GenerateToken(suite, te, Y, H, IDs, csk1, csk2, csk3, ask)
	println(err)

	// ***************************Access phase************************//

	// ****************************User*******************************//
	startTime := time.Now()
	blindToken, r, _ := authentication.GenerateBlindToken_User(suite, pp, tk, y, IDs, te)
	endTime_blindToken := time.Now()
	elapsedTime_blindToken := endTime_blindToken.Sub(startTime)

	// msg1 := []byte("Hello Boneh-Lynn-Shacham1111")
	proof, _ := authentication.GenerateProof_User(suite, pp, y, r)
	endTime_proof := time.Now()
	elapsedTime_proof := endTime_proof.Sub(endTime_blindToken)

	msg := []byte("request data")
	sig, _ := bls.Sign(suite, r, msg)
	endTime_User := time.Now()
	time1 := endTime_User.Sub(startTime)
	// fmt.Println("用户耗时：", time1)

	// ****************************Edge server*******************************//
	err = authentication.Verify(suite, pp, blindToken, proof)
	endTime_verify := time.Now()
	// 计算函数执行时间
	elapsedTime_verify := endTime_verify.Sub(endTime_User)
	// fmt.Println("Verify 耗时", elapsedTime_verify)
	fmt.Println(err)

	result := bls.Verify(suite, blindToken.GetPublic(), msg, sig)
	if result != nil {
		print("BLS fails")
	} else {
		print("BLS success")
	}
	// 获取函数执行结束的时间点
	endTime := time.Now()
	endTime_bls := endTime.Sub(endTime_verify)
	// fmt.Println("验证签名耗时", endTime_bls)

	// 计算函数执行时间
	elapsedTime := endTime.Sub(endTime_User)
	// fmt.Println("边缘服务器耗时", elapsedTime)

	//Audit

	Y_new := authentication.Audit(suite, pp, blindToken, ask, csk2, r)
	// fmt.Print("是否正确恢复出Y：")
	fmt.Println(Y.Equal(Y_new))
	return time1, elapsedTime_blindToken, elapsedTime_proof, elapsedTime_verify, endTime_bls, elapsedTime
}

func AccessTest() {
	// 定义变量用于累加时间值
	var totalVar1, totalVar2, totalVar3, totalVar4, totalVar5, totalVar6 time.Duration
	for i := 0; i < 100; i++ {
		var1, var2, var3, var4, var5, var6 := Authen()
		// 累加时间值
		totalVar1 += var1
		totalVar2 += var2
		totalVar3 += var3
		totalVar4 += var4
		totalVar5 += var5
		totalVar6 += var6

	}
	// 计算平均值
	averageVar1 := totalVar1 / 100
	averageVar2 := totalVar2 / 100
	averageVar3 := totalVar3 / 100
	averageVar4 := totalVar4 / 100
	averageVar5 := totalVar5 / 100
	averageVar6 := totalVar6 / 100

	// 打印结果
	fmt.Printf("用户耗时: %v\n", averageVar1)
	fmt.Printf("计算blindToken耗时: %v\n", averageVar2)
	fmt.Printf("计算proof 耗时: %v\n", averageVar3)
	fmt.Printf("Verify耗时: %v\n", averageVar4)
	fmt.Printf("BLS签名验证耗时: %v\n", averageVar5)
	fmt.Printf("服务器耗时: %v\n", averageVar6)

}

// U: 1.1ms
// CS: 0.8 ms
// AS : 0.2 ms
func Regis() (time.Duration, time.Duration) {

	// init
	suite := bn256.NewSuite()
	csk1, cpk1 := authentication.NewKeyPair(suite, random.New())
	csk2, cpk2 := authentication.NewKeyPair(suite, random.New())
	csk3, cpk3 := authentication.NewKeyPair(suite, random.New())
	ask, apk := authentication.NewKeyPair(suite, random.New())
	_ = authentication.NewPP(cpk1, cpk2, cpk3, apk)

	// user registration
	// // 获取函数开始执行的时间点
	startTime := time.Now()
	te := time.Now().UnixMicro()
	y, Y := authentication.NewKeyPair(suite, random.New())
	yb, _ := y.MarshalBinary()
	print(len(yb))
	// y1, _ := authentication.NewKeyPair(suite, random.New())
	Yb, _ := Y.MarshalBinary()
	u, _ := authentication.Hash2(suite, append(Yb, []byte(fmt.Sprintf("%d", te))...))
	H := u.Mul(y, u)
	Hb, _ := H.MarshalBinary()
	println(len(Hb))
	IDs := "service1:service2:service3:service4"
	// 获取函数执行结束的时间点
	endTime1 := time.Now()
	// 计算函数执行时间
	elapsedTime := endTime1.Sub(startTime)
	_, err := authentication.GenerateToken(suite, te, Y, H, IDs, csk1, csk2, csk3, ask)
	endTime2 := time.Now()

	// 计算函数执行时间
	elapsedTime2 := endTime2.Sub(endTime1)
	fmt.Println("Audit server took", elapsedTime2)
	println(err)
	return elapsedTime, elapsedTime2
}

func RegTest() {
	// 定义变量用于累加时间值
	var totalVar1, totalVar2 time.Duration
	for i := 0; i < 100; i++ {
		var1, var2 := Regis()
		// 累加时间值
		totalVar1 += var1
		totalVar2 += var2

	}
	// 计算平均值
	averageVar1 := totalVar1
	averageVar2 := totalVar2

	// 打印结果
	fmt.Printf("用户耗时: %v\n", averageVar1)
	fmt.Printf("服务器耗时: %v\n", averageVar2)
}

func Comparison_Sok(suite pairing.Suite) {
	point1 := suite.G1().Point().Pick(random.New())
	point2 := suite.G1().Point().Pick(random.New())
	scalar1 := suite.G1().Scalar().Pick(random.New())
	scalar2 := suite.G1().Scalar().Pick(random.New())

	// Mul_G * 45
	for i := 0; i < 45; i++ {
		point1.Mul(scalar1, point1)
	}
	// Add_G * 32
	for i := 0; i < 32; i++ {
		point1.Add(point1, point2)
	}
	// Add_Zp * 16
	for i := 0; i < 10; i++ {
		scalar1.Add(scalar1, scalar2)
	}
	// Mul_zp * 12
	for i := 0; i < 10; i++ {
		scalar1.Mul(scalar1, scalar2)
	}
}

func Comparison_Dual_Token(suite pairing.Suite) {
	//Our scheme
	//Pair
	point1 := suite.G1().Point().Pick(random.New())
	point2 := suite.G1().Point().Pick(random.New())
	point3 := suite.G2().Point().Pick(random.New())
	point4 := suite.G2().Point().Pick(random.New())
	left := suite.Pair(point1, point3)
	right := suite.Pair(point2, point4)
	left.Equal(right)

	scalar1 := suite.G1().Scalar().Pick(random.New())
	bytes, _ := scalar1.MarshalBinary()
	fmt.Println("length", len(bytes))
	scalar2 := suite.G1().Scalar().Pick(random.New())
	//Mul_G * 10
	for i := 0; i < 10; i++ {
		point1.Mul(scalar1, point1)
	}
	// Add_G * 6
	for i := 0; i < 6; i++ {
		point1.Add(point1, point2)
	}
	//Mul_Zp * 2
	for i := 0; i < 2; i++ {
		scalar1.Mul(scalar1, scalar2)
	}
	// Add_Zp * 2
	for i := 0; i < 2; i++ {
		scalar1.Add(scalar1, scalar2)
	}
}

func ComparisonTest() {
	// init
	suite := bn256.NewSuite()
	// 定义变量用于累加时间值
	var sokTime, dualTokenTime time.Duration
	startTime := time.Now()
	for i := 0; i < 100; i++ {
		Comparison_Sok(suite)
	}
	endTime := time.Now()
	sokTime = endTime.Sub(startTime)
	for i := 0; i < 100; i++ {
		Comparison_Dual_Token(suite)
	}
	endTime2 := time.Now()
	dualTokenTime = endTime2.Sub(endTime)

	fmt.Printf("SoK time consumption: %v\n", sokTime)
	fmt.Printf("Dual token time consumption: %v\n", dualTokenTime)
}