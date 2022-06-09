import { Injectable } from '@nestjs/common';
import { User } from 'src/user/user.model';
import { RefreshToken } from './refresh-token.model';
import { v4 as uuid4 } from "uuid";
import { Datastore } from '@google-cloud/datastore';
import { Agent } from 'useragent';


@Injectable()
export class RefreshTokenService {
    private kind = 'RefreshToken'

    constructor(
        private datastore: Datastore
    ) { }

    async createFromUserData(user: User, agent: Agent, userAgent: string): Promise<RefreshToken> {
        const now = new Date()
        const expiredDate = new Date()
        expiredDate.setMonth(expiredDate.getMonth() + 6)
        const refreshToken: RefreshToken = {
            id: uuid4(),
            application: agent.toAgent(),
            platform: agent.os.toString(),
            userAgent: userAgent,
            userId: user.id,
            expiredAt: expiredDate,
            createdAt: now,
        }

        const key = this.datastore.key([this.kind, refreshToken.id])
        const entity = {
            key: key,
            data: [
                {
                    name: 'application',
                    value: refreshToken.application
                },
                {
                    name: 'platform',
                    value: refreshToken.platform
                },
                {
                    name: 'userAgent',
                    value: refreshToken.userAgent,
                    excludeFromIndexes: true
                },
                {
                    name: 'userId',
                    value: refreshToken.userId
                },
                {
                    name: 'expiredAt',
                    value: refreshToken.expiredAt
                },
                {
                    name: 'createdAt',
                    value: refreshToken.createdAt
                }
            ]
        }

        await this.datastore.save(entity)
        return refreshToken
    }
}
