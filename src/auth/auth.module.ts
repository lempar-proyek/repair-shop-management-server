import { Module } from '@nestjs/common';
import { DatastoreModule } from 'src/datastore/datastore.module';
import { UserModule } from 'src/user/user.module';
import { AuthController } from './auth.controller';
import { GoogleAuthService } from './google-auth.service';
import { RefreshTokenService } from './refresh-token/refresh-token.service';
import { AccessTokenService } from './access-token/access-token.service';

@Module({
  imports: [UserModule, DatastoreModule],
  controllers: [AuthController],
  providers: [GoogleAuthService, RefreshTokenService, AccessTokenService]
})
export class AuthModule {}
