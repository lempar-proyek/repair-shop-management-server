import { Body, Controller, Headers, InternalServerErrorException, Post, UnauthorizedException } from '@nestjs/common';
import { LoginDto } from './dto/login.dto';
import { GoogleAuthService } from './google-auth.service';
import { UserService } from 'src/user/user.service';
import { RefreshTokenService } from './refresh-token/refresh-token.service';
import { parse as parseUserAgent, Agent } from "useragent";
import { User } from 'src/user/user.model';
import { AccessTokenService } from './access-token/access-token.service';

@Controller('auth')
export class AuthController {
    constructor(
        private googleAuthService: GoogleAuthService,
        private userService: UserService,
        private refreshTokenService: RefreshTokenService,
        private accessTokenService: AccessTokenService
    ) { }

    @Post("login")
    async loginAndSignup(@Body() loginDto: LoginDto, @Headers('User-Agent') userAgent: string): Promise<object> {
        const ua = parseUserAgent(userAgent)

        switch (loginDto.method.toLowerCase()) {
            case 'google':
                try {
                    const result = await this.googleAuthService.verifyGoogleToken(loginDto.credential)
                    if (result === null) {
                        throw new UnauthorizedException('Unauthorized client')
                    }
                    let foundUser = await this.userService.getUserByGoogleId(result.sub)
                    if (foundUser !== null) {
                        return await this.generateToken(foundUser, ua, userAgent)
                    }
                    return { result }
                } catch (_e) {
                    let e: Error = _e
                    if (e.message.startsWith('Token used too late')) {
                        throw new UnauthorizedException('Token has been expired.')
                    }
                    throw new InternalServerErrorException(e.message)
                }
                break

            default:
                throw new UnauthorizedException('Unknown authentication method')
        }
    }

    private async generateToken(user: User, agent: Agent, userAgent: string): Promise<object> {
        const refreshToken = await this.refreshTokenService.createFromUserData(user, agent, userAgent)
        const accessToken = await this.accessTokenService.createFromRefreshToken(refreshToken)

        return {
            accessToken: JSON.stringify(accessToken),
            expiredIn: 86400,
            type: 'Bearer',
            refreshToken: JSON.stringify(refreshToken)
        }
    }
}
