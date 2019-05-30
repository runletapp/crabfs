import crypto from 'crypto';

import pb from '../protos/key_pb';

export const GenerateKeyPair = async (size: number) => {
    const { publicKey, privateKey } = crypto.generateKeyPairSync('rsa', {
        modulusLength: size,
        publicKeyEncoding: {
            type: 'pkcs1',
            format: 'der',
        },
        privateKeyEncoding: {
            type: 'pkcs1',
            format: 'der',
        },
    });

    const pbPrivateKey = new pb.Key();
    pbPrivateKey.setData(new Uint8Array(privateKey));
    pbPrivateKey.setFormat(pb.Key.Format.X509);

    const pbPublicKey = new pb.Key();
    pbPublicKey.setData(new Uint8Array(publicKey));
    pbPublicKey.setFormat(pb.Key.Format.X509);

    return { privateKey: pbPrivateKey.serializeBinary(), publicKey: pbPublicKey.serializeBinary() }
};