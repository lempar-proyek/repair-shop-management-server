import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { AuthModule } from './auth/auth.module';
import { UserModule } from './user/user.module';
import { DatastoreModule } from './datastore/datastore.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      ignoreEnvFile: true,
      isGlobal: true
    }),
    AuthModule,
    UserModule,
    DatastoreModule
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule { }
