import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { OAuth2Client, TokenPayload } from "google-auth-library";

@Injectable()
export class GoogleAuthService {
    constructor(private configService: ConfigService) { }

    async verifyGoogleToken(gToken: string): Promise<TokenPayload> {
        const serverId = this.configService.get<string>("SERVER_ID")
        const clientId = this.configService.get<string>("CLIENT_ID")
        const client = new OAuth2Client(serverId)
        
        const ticket = await client.verifyIdToken({
            idToken: gToken,
            audience: serverId
        })
        const payload = ticket.getPayload()
        if(payload.azp !== clientId) {
            return null
        }
        return payload
    }
}
