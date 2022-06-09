import { Datastore } from '@google-cloud/datastore';
import { Injectable } from '@nestjs/common';
import { RefreshToken } from '../refresh-token/refresh-token.model';
import { AccessToken } from './access-token.model';
import { v4 as uuid4 } from "uuid";

@Injectable()
export class AccessTokenService {
    private kind = 'AccessToken'

    constructor(
        private datastore: Datastore
    ) { }

    async createFromRefreshToken(refreshToken: RefreshToken): Promise<AccessToken> {
        const now = new Date()
        const expired = new Date()
        expired.setDate(expired.getDate() + 1)
        const accessToken: AccessToken = {
            id: uuid4(),
            userId: refreshToken.userId,
            refreshTokenId: refreshToken.id,
            expiredAt: expired,
            createdAt: now,
        }

        const key = this.datastore.key([this.kind, accessToken.id])
        const data = {
            ...accessToken
        }
        delete data.id

        const entity = {
            key,
            data
        }

        await this.datastore.save(entity)

        return accessToken
    }
}
